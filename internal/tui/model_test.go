package tui

import (
	"reflect"
	"testing"
	"time"

	"github.com/nestorpons/sessions-control/internal/store"
)

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
