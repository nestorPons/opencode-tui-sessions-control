package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

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
	rep := report.Build(report.ViewSessions, segs, "test", 15*time.Minute, 5*time.Minute)
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
