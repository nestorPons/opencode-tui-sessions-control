package domain

import (
	"testing"
	"time"
)

func seg(session, day, project string, startHour, endHour int) Segment {
	loc := time.Local
	d, _ := time.ParseInLocation("2006-01-02", day, loc)
	start := time.Date(d.Year(), d.Month(), d.Day(), startHour, 0, 0, 0, loc)
	end := time.Date(d.Year(), d.Month(), d.Day(), endHour, 0, 0, 0, loc)
	return Segment{
		SessionID: session, Day: day, Project: project, ProjectName: project,
		Start: start, End: end,
		Tokens: Tokens{Input: 100, Output: 10},
	}
}

func TestUniqueDurationMergesOverlaps(t *testing.T) {
	segs := []Segment{
		seg("a", "2026-10-26", "p1", 9, 11),
		seg("b", "2026-10-26", "p1", 10, 12),
		seg("c", "2026-10-26", "p1", 13, 14),
	}
	got := UniqueDuration(segs)
	want := 4 * time.Hour
	if got != want {
		t.Fatalf("UniqueDuration = %v, want %v", got, want)
	}
	if sum := SumDurations(segs); sum != 5*time.Hour {
		t.Fatalf("SumDurations = %v, want 5h", sum)
	}
}

func TestAggregates(t *testing.T) {
	segs := []Segment{
		seg("a", "2026-10-26", "p1", 9, 11),
		seg("b", "2026-09-02", "p2", 9, 10),
		seg("c", "2026-09-03", "p2", 9, 10),
	}
	projects := AggregateByProject(segs)
	if len(projects) != 2 {
		t.Fatalf("projects = %d, want 2", len(projects))
	}
	var p2 *Group
	for i := range projects {
		if projects[i].Label == "p2" {
			p2 = &projects[i]
		}
	}
	if p2 == nil || p2.Sessions != 2 || p2.Unique != 2*time.Hour || p2.Days != 2 {
		t.Fatalf("unexpected p2 aggregate: %+v", projects)
	}
	months := AggregateByMonth(segs)
	if len(months) != 2 || months[0].Key != "2026-10" || months[1].Key != "2026-09" {
		t.Fatalf("unexpected months: %+v", months)
	}
	sessions := AggregateSessions(segs)
	if len(sessions) != 3 || sessions[0].ID != "a" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
}

func TestProjectName(t *testing.T) {
	cases := map[string]string{
		"/home/admin/Proyectos/foo": "foo",
		"/home/admin":               "global",
		"":                          "(sin proyecto)",
	}
	for in, want := range cases {
		if got := ProjectName(in); got != want {
			t.Errorf("ProjectName(%q) = %q, want %q", in, got, want)
		}
	}
}
