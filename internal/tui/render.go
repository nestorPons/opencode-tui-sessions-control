package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nestorpons/sessions-control/internal/report"
)

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("25")).Padding(0, 1)
	styleTab   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Padding(0, 1)
	styleTabOn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("31")).Padding(0, 1)
	styleHead  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
	styleSel   = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("24"))
	styleDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	styleBox   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2)
)

type col struct {
	title string
	width int
	right bool
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Cargando sessions-control..."
	}
	var lines []string
	lines = append(lines, m.headerLines()...)
	switch m.mode {
	case modeFilter:
		lines = append(lines, m.filterModal()...)
	case modeEdit:
		lines = append(lines, m.editModal()...)
	case modeHelp:
		lines = append(lines, m.helpModal()...)
	default:
		lines = append(lines, m.tableLines()...)
	}
	lines = append(lines, m.statusLine())
	lines = append(lines, m.footerLine())

	if pad := m.height - len(lines); pad > 0 {
		lines = append(lines, make([]string, pad)...)
	}
	max := lipgloss.NewStyle().MaxWidth(m.width)
	for i, l := range lines {
		lines[i] = max.Render(l)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) headerLines() []string {
	tabs := []struct {
		key   string
		label string
		view  report.View
	}{
		{"1", "Sesiones", report.ViewSessions},
		{"2", "Proyectos", report.ViewProjects},
		{"3", "Meses", report.ViewMonths},
		{"4", "Días", report.ViewDays},
	}
	var tabParts []string
	for _, t := range tabs {
		label := t.key + " " + t.label
		if m.view == t.view {
			tabParts = append(tabParts, styleTabOn.Render(label))
		} else {
			tabParts = append(tabParts, styleTab.Render(label))
		}
	}
	left := styleTitle.Render("Sessions Control") + " " + strings.Join(tabParts, "")
	right := ""
	if !m.stats.SyncedAt.IsZero() {
		right = styleDim.Render("sync " + m.stats.SyncedAt.Format("02-01 15:04"))
	}
	first := left
	if gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 1; gap > 0 {
		first = left + strings.Repeat(" ", gap) + right
	}
	filterLine := styleDim.Render("Filtro: ") + m.filter.Describe() +
		styleDim.Render("   ·   ") + m.rep.Summary
	source := m.stats.Source
	if source == "" {
		source = "(sin sincronizar)"
	}
	sourceLine := styleDim.Render("Fuente: " + source)
	return []string{first, filterLine, sourceLine}
}

func (m *Model) bodyRows() int {
	n := m.height - 9
	if n < 3 {
		n = 3
	}
	return n
}

func (m *Model) tableLines() []string {
	cols := m.columns()
	header := renderRow(cols, headerCells(cols), false)
	out := []string{styleHead.Render(padLine(header, m.width))}

	start := m.offset
	end := start + m.bodyRows()
	if end > len(m.rep.Rows) {
		end = len(m.rep.Rows)
	}
	for i := start; i < end; i++ {
		line := renderRow(cols, m.rep.Rows[i], false)
		if i == m.cursor {
			out = append(out, styleSel.Render(padLine(line, m.width)))
		} else {
			out = append(out, padLine(line, m.width))
		}
	}
	if len(m.rep.Rows) == 0 {
		out = append(out, styleDim.Render("  (sin datos para el filtro actual)"))
	}
	for len(out) < 1+m.bodyRows() {
		out = append(out, "")
	}
	return out
}

func (m *Model) columns() []col {
	avail := m.width - 4
	if avail < 50 {
		avail = 50
	}
	var weights []float64
	var titles []string
	var rights []bool
	switch m.view {
	case report.ViewProjects:
		titles = []string{"Proyecto", "Inicio", "Fin", "Ses.", "Días", "Horas", "Único", "Tokens", "Coste"}
		weights = []float64{5, 1.5, 1.5, 1.6, 1.2, 1.6, 1.6, 2.3, 1.8}
		rights = []bool{false, false, false, true, true, true, true, true, true}
	case report.ViewMonths:
		titles = []string{"Mes", "Ses.", "Días", "Horas", "Único", "Tokens", "Coste"}
		weights = []float64{3, 2, 1.6, 2, 2, 2.6, 2}
		rights = []bool{false, true, true, true, true, true, true}
	case report.ViewDays:
		titles = []string{"Fecha", "Ses.", "Horas", "Único", "Tokens", "Coste"}
		weights = []float64{3, 2, 2, 2, 2.6, 2}
		rights = []bool{false, true, true, true, true, true}
	default:
		titles = []string{"Fecha", "Inicio", "Fin", "Horas", "Proyecto", "Sesión", "Tokens", "Coste"}
		weights = []float64{2.2, 1.4, 1.4, 1.4, 3.6, 10, 2.2, 1.8}
		rights = []bool{false, false, false, true, false, false, true, true}
	}
	var total float64
	for _, w := range weights {
		total += w
	}
	cols := make([]col, len(titles))
	used := 0
	for i := range titles {
		min := len(titles[i]) + 1
		if min < 5 {
			min = 5
		}
		w := int(float64(avail-len(titles)+1) * weights[i] / total)
		if w < min {
			w = min
		}
		cols[i] = col{title: titles[i], width: w, right: rights[i]}
		used += w
	}
	used += len(cols) - 1
	if used > avail {
		excess := used - avail
		for i := range cols {
			if excess == 0 {
				break
			}
			room := cols[i].width - (len(cols[i].title) + 1)
			if room <= 0 {
				continue
			}
			cut := room
			if cut > excess {
				cut = excess
			}
			cols[i].width -= cut
			excess -= cut
		}
	}
	return cols
}

func headerCells(cols []col) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.title
	}
	return out
}

func renderRow(cols []col, cells []string, _ bool) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		v := ""
		if i < len(cells) {
			v = cells[i]
		}
		parts[i] = fit(v, c.width, c.right)
	}
	return strings.Join(parts, " ")
}

func fit(s string, width int, right bool) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) > width {
		if width == 1 {
			r = r[:1]
		} else {
			r = append(r[:width-1], '…')
		}
	}
	s = string(r)
	if right {
		pad := width - len([]rune(s))
		if pad > 0 {
			return strings.Repeat(" ", pad) + s
		}
		return s
	}
	pad := width - len([]rune(s))
	if pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func padLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateLine(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > width {
		r = r[:len(r)-1]
	}
	return string(r)
}

func (m *Model) statusLine() string {
	text := ""
	switch {
	case m.busy != "":
		text = styleOK.Render(m.busy)
	case m.isError:
		text = styleErr.Render(m.status)
	case m.status != "":
		text = styleOK.Render(m.status)
	default:
		text = styleDim.Render("Listo")
	}
	n := countSessions(m.rep)
	label := "sesiones"
	if n == 1 {
		label = "sesión"
	}
	info := styleDim.Render(fmt.Sprintf("  %d filas · %d %s · %s", len(m.rep.Rows), n, label, m.rep.Filter))
	return padLine(" "+text+info, m.width)
}

func countSessions(r report.Report) int {
	seen := map[string]struct{}{}
	for _, s := range r.Raw {
		seen[s.SessionID] = struct{}{}
	}
	return len(seen)
}

func (m *Model) footerLine() string {
	keys := "1-4 vistas · ↑↓ mover · f filtro · t/y/w/m fechas · e editar · r restaurar · s sync · x Excel · p PDF · ? ayuda · q salir"
	return styleDim.Render(padLine(" "+keys, m.width))
}

func (m *Model) filterModal() []string {
	boxWidth := m.width - 8
	if boxWidth > 100 {
		boxWidth = 100
	}
	if boxWidth < 40 {
		boxWidth = 40
	}
	lines := []string{
		styleTitle.Render("Filtrar sesiones"),
		"",
		m.filterInput.View(),
		"",
		styleDim.Render("Días: 2026-10-26, 2026-10-27   ·   Mes: month=2026-10   ·   Proyecto: project=clowww.es"),
		styleDim.Render("Rango: from=2026-10-01 to=2026-10-27   ·   Atajos: ayer · semana · all"),
		"",
		"Enter aplicar   ·   Esc cancelar",
	}
	box := styleBox.Width(boxWidth).Render(strings.Join(lines, "\n"))
	return strings.Split(box, "\n")
}

func (m *Model) editModal() []string {
	boxWidth := m.width - 8
	if boxWidth > 100 {
		boxWidth = 100
	}
	if boxWidth < 50 {
		boxWidth = 50
	}
	title := truncateLine(m.editTarget.Title, boxWidth-8)
	lines := []string{
		styleTitle.Render("Editar horas"),
		"",
		fmt.Sprintf("Día: %s   ·   Proyecto: %s", m.editTarget.Day, m.editTarget.ProjectName),
		styleDim.Render("Sesión: " + title),
		"",
		m.editStart.View(),
		m.editEnd.View(),
		"",
		styleDim.Render("Formato HH:MM o HH:MM:SS. Tab cambia de campo."),
		"Enter guardar   ·   Esc cancelar",
	}
	box := styleBox.Width(boxWidth).Render(strings.Join(lines, "\n"))
	return strings.Split(box, "\n")
}

func (m *Model) helpModal() []string {
	lines := []string{
		styleTitle.Render("Ayuda"),
		"",
		"1 / 2 / 3 / 4    Vistas: sesiones, proyectos, meses, días",
		"↑ ↓ j k          Mover selección",
		"PgUp / PgDn      Página arriba / abajo",
		"g / G            Ir al principio / final",
		"",
		"f                Filtrar por días, mes, rango o proyecto",
		"t / y / w / m    Filtro rápido sobre lo visible: hoy, ayer, 7 días, mes",
		"c                Quitar el filtro",
		"e                Editar inicio/fin (vista Sesiones)",
		"r                Restaurar las horas originales",
		"s                Sincronizar con opencode.db",
		"x                Exportar la vista actual a Excel",
		"p                Exportar la vista actual a PDF",
		"",
		"El filtro acepta: 2026-10-26, 2026-10-27 · month=2026-10 ·",
		"project=clowww.es · from=2026-10-01 to=2026-10-27 · ayer · semana · all",
		"",
		"Esc / ? / Enter  cerrar ayuda",
	}
	box := styleBox.Width(96).Render(strings.Join(lines, "\n"))
	return strings.Split(box, "\n")
}
