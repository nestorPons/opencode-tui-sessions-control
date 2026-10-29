package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nestorpons/sessions-control/internal/config"
	"github.com/nestorpons/sessions-control/internal/domain"
	"github.com/nestorpons/sessions-control/internal/export"
	"github.com/nestorpons/sessions-control/internal/report"
	"github.com/nestorpons/sessions-control/internal/store"
)

type mode int

const (
	modeList mode = iota
	modeFilter
	modeEdit
	modeHelp
)

type Model struct {
	st          *store.Store
	cfg         config.Config
	stats       store.Stats
	filter      store.Filter
	view        report.View
	rep         report.Report
	width       int
	height      int
	cursor      int
	offset      int
	mode        mode
	filterInput textinput.Model
	editStart   textinput.Model
	editEnd     textinput.Model
	editFocus   int
	editTarget  domain.Segment
	status      string
	isError     bool
	busy        string
}

type syncMsg struct {
	stats store.Stats
	err   error
}

type exportMsg struct {
	path string
	err  error
}

func New(st *store.Store, cfg config.Config, stats store.Stats) *Model {
	fi := textinput.New()
	fi.Placeholder = "2026-10-26, 2026-10-27 · month=2026-10 · project=clowww.es · all"
	fi.CharLimit = 200
	fi.Width = 72
	fi.Prompt = "> "

	es := textinput.New()
	es.Placeholder = "HH:MM:SS"
	es.CharLimit = 19
	es.Width = 12
	es.Prompt = "Inicio: "

	ee := textinput.New()
	ee.Placeholder = "HH:MM:SS"
	ee.CharLimit = 19
	ee.Width = 12
	ee.Prompt = "Fin:    "

	m := &Model{
		st:          st,
		cfg:         cfg,
		stats:       stats,
		width:       100,
		height:      30,
		view:        report.ViewSessions,
		filter:      store.Filter{},
		filterInput: fi,
		editStart:   es,
		editEnd:     ee,
	}
	m.reload()
	m.cursor = 0
	m.clamp()
	return m
}

func (m *Model) reload() {
	segs, err := m.st.Segments(m.filter)
	if err != nil {
		m.status = "error cargando datos: " + err.Error()
		m.isError = true
		return
	}
	m.rep = report.Build(m.view, segs, m.filter.Describe())
	if st, err := m.st.Stats(); err == nil {
		m.stats = st
	}
	m.clamp()
}

func (m *Model) selectKey(key string) {
	if key == "" {
		return
	}
	for i, s := range m.rep.Raw {
		if s.SessionID+"|"+s.Day == key {
			m.cursor = i
			return
		}
	}
}

func (m *Model) clamp() {
	if m.cursor >= len(m.rep.Rows) {
		m.cursor = len(m.rep.Rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	rows := m.bodyRows()
	if rows < 1 {
		rows = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	maxOffset := len(m.rep.Rows) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ensureVisible()
		return m, nil
	case syncMsg:
		m.busy = ""
		if msg.err != nil {
			m.status = "error al sincronizar: " + msg.err.Error()
			m.isError = true
			return m, nil
		}
		m.stats = msg.stats
		m.reload()
		m.status = fmt.Sprintf("Sincronizado: %d sesiones, %d tramos", msg.stats.Sessions, msg.stats.Segments)
		m.isError = false
		return m, nil
	case exportMsg:
		m.busy = ""
		if msg.err != nil {
			m.status = "error al exportar: " + msg.err.Error()
			m.isError = true
			return m, nil
		}
		m.status = "Exportado: " + msg.path
		m.isError = false
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeFilter {
			return m.updateFilter(msg)
		}
		if m.mode == modeEdit {
			return m.updateEdit(msg)
		}
		if m.mode == modeHelp {
			switch msg.String() {
			case "esc", "?", "q", "enter":
				m.mode = modeList
			}
			return m, nil
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
		return m, nil
	case "1":
		m.setView(report.ViewSessions)
	case "2":
		m.setView(report.ViewProjects)
	case "3":
		m.setView(report.ViewMonths)
	case "4":
		m.setView(report.ViewDays)
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup", "ctrl+b":
		m.cursor -= m.bodyRows()
	case "pgdown", "ctrl+f":
		m.cursor += m.bodyRows()
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.rep.Rows) - 1
	case "f":
		m.mode = modeFilter
		m.filterInput.SetValue(m.filter.InputString())
		m.filterInput.CursorEnd()
		m.filterInput.Focus()
		return m, textinput.Blink
	case "t":
		m.applyDatePreset("hoy")
	case "y":
		m.applyDatePreset("ayer")
	case "w":
		m.applyDatePreset("semana")
	case "m":
		m.applyDatePreset("mes")
	case "c":
		m.filter = store.Filter{}
		m.reload()
		m.cursor = 0
		m.status = "Filtro eliminado"
		m.isError = false
	case "s":
		m.busy = "Sincronizando con opencode.db..."
		return m, m.syncCmd()
	case "e":
		m.startEdit()
		return m, nil
	case "r":
		m.resetTimes()
		return m, nil
	case "x":
		m.busy = "Generando Excel..."
		return m, m.exportCmd("xlsx")
	case "p":
		m.busy = "Generando PDF..."
		return m, m.exportCmd("pdf")
	}
	m.clamp()
	return m, nil
}

func (m *Model) setView(v report.View) {
	if m.view == v {
		return
	}
	m.view = v
	m.cursor = 0
	m.offset = 0
	m.reload()
}

func (m *Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.filterInput.Blur()
		return m, nil
	case "enter":
		f, err := parseFilter(m.filterInput.Value())
		if err != nil {
			m.status = err.Error()
			m.isError = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}
		m.filter = f
		m.mode = modeList
		m.filterInput.Blur()
		m.cursor = 0
		m.offset = 0
		m.reload()
		m.status = "Filtro: " + m.filter.Describe()
		m.isError = false
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *Model) startEdit() {
	if m.view != report.ViewSessions {
		m.status = "La edición de horas solo está disponible en la vista Sesiones (1)"
		m.isError = true
		return
	}
	if len(m.rep.Raw) == 0 || m.cursor >= len(m.rep.Raw) {
		m.status = "No hay ninguna sesión seleccionada"
		m.isError = true
		return
	}
	m.editTarget = m.rep.Raw[m.cursor]
	m.editStart.SetValue(m.editTarget.Start.Format("15:04:05"))
	m.editEnd.SetValue(m.editTarget.End.Format("15:04:05"))
	m.editStart.CursorEnd()
	m.editEnd.CursorEnd()
	m.editFocus = 0
	m.editStart.Focus()
	m.editEnd.Blur()
	m.mode = modeEdit
	m.status = ""
	m.isError = false
}

func (m *Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.editStart.Blur()
		m.editEnd.Blur()
		return m, nil
	case "tab", "down":
		m.editFocus = (m.editFocus + 1) % 2
		m.applyEditFocus()
		return m, textinput.Blink
	case "shift+tab", "up":
		m.editFocus = (m.editFocus + 1) % 2
		m.applyEditFocus()
		return m, textinput.Blink
	case "enter":
		start, err := parseClock(m.editTarget.Day, m.editStart.Value())
		if err != nil {
			m.status = "inicio: " + err.Error()
			m.isError = true
			return m, nil
		}
		end, err := parseClock(m.editTarget.Day, m.editEnd.Value())
		if err != nil {
			m.status = "fin: " + err.Error()
			m.isError = true
			return m, nil
		}
		if !end.After(start) {
			m.status = "la hora de fin debe ser posterior a la de inicio"
			m.isError = true
			return m, nil
		}
		if err := m.st.UpdateTimes(m.editTarget.SessionID, m.editTarget.Day, start, end); err != nil {
			m.status = err.Error()
			m.isError = true
			return m, nil
		}
		key := m.editTarget.SessionID + "|" + m.editTarget.Day
		m.mode = modeList
		m.editStart.Blur()
		m.editEnd.Blur()
		m.reload()
		m.selectKey(key)
		m.ensureVisible()
		m.status = fmt.Sprintf("Horas actualizadas: %s · %s-%s", m.editTarget.Day,
			start.Format("15:04"), end.Format("15:04"))
		m.isError = false
		return m, nil
	}
	var cmd tea.Cmd
	if m.editFocus == 0 {
		m.editStart, cmd = m.editStart.Update(msg)
	} else {
		m.editEnd, cmd = m.editEnd.Update(msg)
	}
	return m, cmd
}

func (m *Model) applyEditFocus() {
	if m.editFocus == 0 {
		m.editStart.Focus()
		m.editEnd.Blur()
	} else {
		m.editEnd.Focus()
		m.editStart.Blur()
	}
}

func (m *Model) resetTimes() {
	if m.view != report.ViewSessions || len(m.rep.Raw) == 0 || m.cursor >= len(m.rep.Raw) {
		m.status = "Selecciona una sesión en la vista Sesiones (1)"
		m.isError = true
		return
	}
	s := m.rep.Raw[m.cursor]
	if !s.Edited {
		m.status = "Esta sesión no tiene horas editadas"
		m.isError = false
		return
	}
	if err := m.st.ResetTimes(s.SessionID, s.Day); err != nil {
		m.status = err.Error()
		m.isError = true
		return
	}
	key := s.SessionID + "|" + s.Day
	m.reload()
	m.selectKey(key)
	m.ensureVisible()
	m.status = "Horas restauradas al valor original"
	m.isError = false
}

func (m *Model) syncCmd() tea.Cmd {
	st, cfg := m.st, m.cfg
	return func() tea.Msg {
		stats, err := st.Sync(cfg.SourceDB)
		return syncMsg{stats: stats, err: err}
	}
}

func (m *Model) exportCmd(format string) tea.Cmd {
	rep, cfg, stats, view := m.rep, m.cfg, m.stats, m.view
	ext := format
	if format == "xlsx" {
		ext = "xlsx"
	}
	name := fmt.Sprintf("sessions-control_%s_%s.%s", view, time.Now().Format("20060102-150405"), ext)
	path := filepath.Join(cfg.ExportsDir, name)
	return func() tea.Msg {
		var err error
		if format == "pdf" {
			err = export.PDF(path, rep, stats)
		} else {
			err = export.Excel(path, rep, stats)
		}
		return exportMsg{path: path, err: err}
	}
}

func (m *Model) applyDatePreset(kind string) {
	f, ok := datePreset(kind, time.Now(), m.filter)
	if !ok {
		return
	}
	m.filter = f
	m.cursor = 0
	m.offset = 0
	m.reload()
	m.status = "Filtro " + kind + ": " + m.filter.Describe()
	m.isError = false
}

func datePreset(kind string, now time.Time, base store.Filter) (store.Filter, bool) {
	f := store.Filter{Project: base.Project}
	anchor := dateAnchor(base, now)
	switch kind {
	case "hoy":
		f.Days = []string{anchor.Format("2006-01-02")}
	case "ayer":
		f.Days = []string{anchor.AddDate(0, 0, -1).Format("2006-01-02")}
	case "semana":
		f.From = anchor.AddDate(0, 0, -6).Format("2006-01-02")
		f.To = anchor.Format("2006-01-02")
	case "mes":
		first := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, anchor.Location())
		f.From = first.Format("2006-01-02")
		f.To = first.AddDate(0, 1, -1).Format("2006-01-02")
	default:
		return base, false
	}
	return f, true
}

func dateAnchor(base store.Filter, now time.Time) time.Time {
	loc := now.Location()
	if base.Month != "" {
		if t, err := time.ParseInLocation("2006-01", base.Month, loc); err == nil {
			return t.AddDate(0, 1, -1)
		}
	}
	if base.To != "" {
		if t, err := time.ParseInLocation("2006-01-02", base.To, loc); err == nil {
			return t
		}
	}
	if base.From != "" {
		if t, err := time.ParseInLocation("2006-01-02", base.From, loc); err == nil {
			return t
		}
	}
	var latest time.Time
	for _, d := range base.Days {
		if t, err := time.ParseInLocation("2006-01-02", d, loc); err == nil && t.After(latest) {
			latest = t
		}
	}
	if !latest.IsZero() {
		return latest
	}
	return now
}

func parseClock(day, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("hora vacía")
	}
	if strings.ContainsAny(value, " -") {
		for _, l := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
			if t, err := time.ParseInLocation(l, value, time.Local); err == nil {
				return t, nil
			}
		}
	} else {
		for _, l := range []string{"15:04:05", "15:04"} {
			if t, err := time.ParseInLocation("2006-01-02 "+l, day+" "+value, time.Local); err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("formato no válido %q (usa HH:MM o HH:MM:SS)", value)
}

func parseFilter(input string) (store.Filter, error) {
	var f store.Filter
	input = strings.TrimSpace(input)
	if input == "" || strings.EqualFold(input, "all") || strings.EqualFold(input, "todo") || input == "*" {
		return f, nil
	}
	tokens := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	today := time.Now()
	for _, tok := range tokens {
		low := strings.ToLower(tok)
		switch {
		case strings.HasPrefix(low, "month=") || strings.HasPrefix(low, "mes="):
			v := tok[strings.Index(tok, "=")+1:]
			if _, err := time.Parse("2006-01", v); err != nil {
				return f, fmt.Errorf("mes no válido %q (usa YYYY-MM)", v)
			}
			f.Month = v
		case strings.HasPrefix(low, "project=") || strings.HasPrefix(low, "proyecto="):
			f.Project = tok[strings.Index(tok, "=")+1:]
		case strings.HasPrefix(low, "from=") || strings.HasPrefix(low, "desde="):
			v := tok[strings.Index(tok, "=")+1:]
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return f, fmt.Errorf("fecha desde no válida %q", v)
			}
			f.From = v
		case strings.HasPrefix(low, "to=") || strings.HasPrefix(low, "hasta="):
			v := tok[strings.Index(tok, "=")+1:]
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return f, fmt.Errorf("fecha hasta no válida %q", v)
			}
			f.To = v
		case low == "hoy" || low == "today":
			f.Days = append(f.Days, today.Format("2006-01-02"))
		case low == "ayer" || low == "yesterday":
			f.Days = append(f.Days, today.AddDate(0, 0, -1).Format("2006-01-02"))
		case low == "semana" || low == "week":
			f.From = today.AddDate(0, 0, -6).Format("2006-01-02")
			f.To = today.Format("2006-01-02")
		default:
			if _, err := time.Parse("2006-01-02", tok); err == nil {
				f.Days = append(f.Days, tok)
				continue
			}
			if _, err := time.Parse("2006-01", tok); err == nil {
				f.Month = tok
				continue
			}
			return f, fmt.Errorf("filtro no reconocido %q", tok)
		}
	}
	return f, nil
}
