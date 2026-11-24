package tui

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nestorpons/sessions-control/internal/config"
	"github.com/nestorpons/sessions-control/internal/domain"
	"github.com/nestorpons/sessions-control/internal/report"
	"github.com/nestorpons/sessions-control/internal/store"
)

func TestShiftDays(t *testing.T) {
	tests := []struct {
		name  string
		base  store.Filter
		delta int
		want  store.Filter
		ok    bool
	}{
		{"ayer atrás", store.Filter{Days: []string{"2026-10-26"}}, -1,
			store.Filter{Days: []string{"2026-10-25"}}, true},
		{"ayer adelante", store.Filter{Days: []string{"2026-10-26"}}, 1,
			store.Filter{Days: []string{"2026-10-27"}}, true},
		{"cambio de mes", store.Filter{Days: []string{"2026-10-01"}}, -1,
			store.Filter{Days: []string{"2026-09-30"}}, true},
		{"cambio de año", store.Filter{Days: []string{"2026-01-01"}}, -1,
			store.Filter{Days: []string{"2025-12-31"}}, true},
		{"varios días", store.Filter{Days: []string{"2026-10-26", "2026-10-27"}}, -1,
			store.Filter{Days: []string{"2026-10-25", "2026-10-26"}}, true},
		{"conserva proyecto", store.Filter{Days: []string{"2026-10-26"}, Project: "alpha"}, 1,
			store.Filter{Days: []string{"2026-10-27"}, Project: "alpha"}, true},
		{"rango", store.Filter{From: "2026-10-21", To: "2026-10-27"}, -1,
			store.Filter{From: "2026-10-20", To: "2026-10-26"}, true},
		{"solo mes", store.Filter{Month: "2026-10"}, 1, store.Filter{Month: "2026-10"}, false},
		{"sin fechas", store.Filter{}, 1, store.Filter{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := shiftDays(tt.base, tt.delta)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDatePreset(t *testing.T) {
	now := time.Date(2026, 10, 27, 15, 0, 0, 0, time.Local)

	tests := []struct {
		name string
		base store.Filter
		kind string
		want store.Filter
		ok   bool
	}{
		{"hoy sin filtro", store.Filter{}, "hoy", store.Filter{Days: []string{"2026-10-27"}}, true},
		{"ayer sin filtro", store.Filter{}, "ayer", store.Filter{Days: []string{"2026-10-26"}}, true},
		{"semana sin filtro", store.Filter{}, "semana", store.Filter{From: "2026-10-21", To: "2026-10-27"}, true},
		{"mes sin filtro", store.Filter{}, "mes", store.Filter{From: "2026-10-01", To: "2026-10-31"}, true},
		{"mes visible", store.Filter{Project: "alpha", Month: "2026-09"}, "mes",
			store.Filter{From: "2026-09-01", To: "2026-09-30", Project: "alpha"}, true},
		{"hoy visible", store.Filter{Project: "alpha", Month: "2026-09"}, "hoy",
			store.Filter{Days: []string{"2026-09-30"}, Project: "alpha"}, true},
		{"rango visible", store.Filter{From: "2026-05-10", To: "2026-05-20"}, "mes",
			store.Filter{From: "2026-05-01", To: "2026-05-31"}, true},
		{"días visibles", store.Filter{Days: []string{"2026-03-02", "2026-03-05"}}, "mes",
			store.Filter{From: "2026-03-01", To: "2026-03-31"}, true},
		{"desconocido", store.Filter{Month: "2026-09"}, "otro", store.Filter{Month: "2026-09"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := datePreset(tt.kind, now, tt.base)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestViewShowsEffectiveColumn(t *testing.T) {
	day := time.Date(2026, 11, 23, 11, 14, 0, 0, time.Local)
	segs := []domain.Segment{{
		SessionID: "s1", Day: "2026-11-23", Project: "/tmp/p", ProjectName: "p",
		Title: "una sesión", Start: day, End: day.Add(2 * time.Hour),
		Effective: 90 * time.Minute, Allocated: 60 * time.Minute,
	}}
	rep := report.Build(report.ViewSessions, segs, "test", 15*time.Minute, 5*time.Minute, 19)
	m := &Model{
		width: 140, height: 30, view: report.ViewSessions, rep: rep,
		stats: store.Stats{EffectiveGap: 15 * time.Minute, DayGap: 5 * time.Minute},
	}
	out := m.View()
	if !strings.Contains(out, "Efectivo") {
		t.Fatalf("view has no Efectivo column:\n%s", out)
	}
	if !strings.Contains(out, "Reparto") {
		t.Fatalf("view has no Reparto column:\n%s", out)
	}
	if !strings.Contains(out, "huecos ≤ 15 min") {
		t.Fatalf("view has no effective gap note:\n%s", out)
	}
	if !strings.Contains(out, "1.50") || !strings.Contains(out, "1.00") {
		t.Fatalf("view has no effective/allocated value:\n%s", out)
	}
}

func TestToggleInternalBlocksSystemProjects(t *testing.T) {
	m := &Model{
		view:   report.ViewProjects,
		cursor: 0,
		rep: report.Report{Raw: []domain.Segment{{
			Project: "/home/admin", ProjectName: "global",
		}}},
	}
	if _, cmd := m.toggleInternal(); cmd != nil {
		t.Fatal("global must not trigger a sync")
	}
	if !m.isError || m.status != "global es siempre interno" {
		t.Fatalf("status = %q, error = %v", m.status, m.isError)
	}
}

func TestParseRate(t *testing.T) {
	cases := map[string]float64{
		"19":         19,
		"19,5":       19.5,
		"19.50":      19.5,
		"19,50 €/h":  19.5,
		" 22.75 /h ": 22.75,
	}
	for in, want := range cases {
		got, err := parseRate(in)
		if err != nil || got != want {
			t.Fatalf("parseRate(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "-1", "abc"} {
		if _, err := parseRate(in); err == nil {
			t.Fatalf("parseRate(%q) should fail", in)
		}
	}
}

func TestViewShowsImporteColumn(t *testing.T) {
	day := time.Date(2026, 11, 23, 9, 0, 0, 0, time.Local)
	segs := []domain.Segment{{
		SessionID: "s1", Day: "2026-11-23", Project: "/tmp/p", ProjectName: "p",
		Title: "una sesión", Start: day, End: day.Add(4 * time.Hour),
		Effective: 4 * time.Hour, Allocated: 4 * time.Hour,
	}}
	rep := report.Build(report.ViewProjects, segs, "test", 15*time.Minute, 5*time.Minute, 19)
	m := &Model{
		width: 160, height: 30, view: report.ViewProjects, rep: rep, rate: 19,
		stats: store.Stats{EffectiveGap: 15 * time.Minute, DayGap: 5 * time.Minute, Rate: 19},
	}
	out := m.View()
	if !strings.Contains(out, "Importe") || !strings.Contains(out, "76,00") {
		t.Fatalf("view has no Importe column/value:\n%s", out)
	}
	if !strings.Contains(out, "Tarifa: 19,00 €/h") {
		t.Fatalf("view has no rate note:\n%s", out)
	}
}

func TestRateEditorPersists(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := New(st, config.Config{}, store.Stats{EffectiveGap: 15 * time.Minute, DayGap: 5 * time.Minute, Rate: 19})
	m.startRate()
	if m.mode != modeRate {
		t.Fatal("rate editor did not open")
	}
	m.rateInput.SetValue("22,50")
	m.updateRate(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeList || m.rate != 22.5 {
		t.Fatalf("mode = %v, rate = %v", m.mode, m.rate)
	}
	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rate != 22.5 {
		t.Fatalf("stored rate = %v, want 22.5", stats.Rate)
	}
}
