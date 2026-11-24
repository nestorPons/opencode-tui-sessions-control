package report

import (
	"testing"
	"time"

	"github.com/nestorpons/sessions-control/internal/domain"
)

func TestBuildImporte(t *testing.T) {
	day := time.Date(2026, 11, 23, 9, 0, 0, 0, time.Local)
	segs := []domain.Segment{{
		SessionID: "s1", Day: "2026-11-23", Project: "/tmp/p", ProjectName: "p",
		Title: "una sesión", Start: day, End: day.Add(4 * time.Hour),
		Effective: 4 * time.Hour, Allocated: 4 * time.Hour,
	}}
	rep := Build(ViewProjects, segs, "test", 15*time.Minute, 5*time.Minute, 19)

	idx := -1
	for i, h := range rep.Headers {
		if h == "Importe" {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("no Importe column")
	}
	if got := rep.Rows[0][idx]; got != "76,00 €" {
		t.Fatalf("row importe = %q, want 76,00 €", got)
	}
	if got := rep.Data[0][idx].(float64); got != 76 {
		t.Fatalf("data importe = %v, want 76", got)
	}
	if got := rep.Totals[idx]; got != "76,00 €" {
		t.Fatalf("total importe = %q, want 76,00 €", got)
	}
}

func TestRowWidths(t *testing.T) {
	day := time.Date(2026, 11, 23, 9, 0, 0, 0, time.Local)
	segs := []domain.Segment{
		{SessionID: "s1", Day: "2026-11-23", Project: "/tmp/a", ProjectName: "a",
			Title: "uno", Start: day, End: day.Add(4 * time.Hour),
			Effective: 4 * time.Hour, Allocated: 4 * time.Hour},
		{SessionID: "s2", Day: "2026-11-23", Project: "/tmp/b", ProjectName: "b",
			Title: "dos", Start: day.Add(4 * time.Hour), End: day.Add(6 * time.Hour),
			Effective: 2 * time.Hour, Allocated: 2 * time.Hour},
	}
	for _, v := range []View{ViewSessions, ViewProjects, ViewMonths, ViewDays} {
		rep := Build(v, segs, "test", 15*time.Minute, 5*time.Minute, 19)
		n := len(rep.Headers)
		for i, row := range rep.Rows {
			if len(row) != n {
				t.Fatalf("%s row %d has %d cells, want %d", v, i, len(row), n)
			}
		}
		for i, row := range rep.Data {
			if len(row) != n {
				t.Fatalf("%s data %d has %d cells, want %d", v, i, len(row), n)
			}
		}
		if len(rep.Totals) != n {
			t.Fatalf("%s totals have %d cells, want %d", v, len(rep.Totals), n)
		}
		if len(rep.TotalData) != n {
			t.Fatalf("%s total data has %d cells, want %d", v, len(rep.TotalData), n)
		}
	}
}
