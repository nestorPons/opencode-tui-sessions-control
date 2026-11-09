package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func createSource(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT, title TEXT, time_created INTEGER, time_updated INTEGER)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, data TEXT)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	day := func(h, m int) int64 {
		return time.Date(2026, 10, 26, h, m, 0, 0, time.Local).UnixMilli()
	}
	msgs := []struct {
		session   string
		ts        int64
		completed int64
		tokIn     int
		tokOut    int
	}{
		{"s1", day(9, 0), day(9, 10), 100, 20},
		{"s1", day(9, 20), day(9, 30), 50, 10},
		{"s2", day(10, 0), day(10, 30), 200, 40},
		{"s3", day(11, 0), day(11, 5), 0, 0},
	}
	for i, m := range msgs {
		data := fmt.Sprintf(`{"role":"assistant","cost":0.1,"tokens":{"input":%d,"output":%d,"reasoning":0,"cache":{"read":0,"write":0}},"time":{"created":%d,"completed":%d}}`,
			m.tokIn, m.tokOut, m.ts, m.completed)
		if _, err := db.Exec(`INSERT INTO message(id,session_id,time_created,data) VALUES(?,?,?,?)`,
			fmt.Sprintf("m%d", i), m.session, m.ts, data); err != nil {
			t.Fatal(err)
		}
	}
	sessions := []struct {
		id, dir string
		created int64
	}{
		{"s1", "/home/admin/Proyectos/alpha", day(9, 0)},
		{"s2", "/home/admin/Proyectos/beta", day(10, 0)},
		{"s3", "/home/admin", day(11, 0)},
	}
	for _, s := range sessions {
		if _, err := db.Exec(`INSERT INTO session(id,directory,title,time_created,time_updated) VALUES(?,?,?,?,?)`,
			s.id, s.dir, "title "+s.id, s.created, s.created); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSyncQueryAndOverrides(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	createSource(t, src)
	st, err := New(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	stats, err := st.Sync(src)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sessions != 3 || stats.Segments != 3 {
		t.Fatalf("stats = %+v", stats)
	}

	segs, err := st.Segments(Filter{Days: []string{"2026-10-26"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 3 {
		t.Fatalf("segments = %d, want 3", len(segs))
	}
	var total int64
	for _, s := range segs {
		total += s.Tokens.Total()
		if s.ProjectName == "" {
			t.Fatalf("empty project name: %+v", s)
		}
	}
	if total != 420 {
		t.Fatalf("tokens = %d, want 420", total)
	}

	target := segs[0]
	start := target.Start.Add(15 * time.Minute)
	end := target.End.Add(30 * time.Minute)
	if err := st.UpdateTimes(target.SessionID, target.Day, start, end); err != nil {
		t.Fatal(err)
	}
	edited, err := st.Segments(Filter{Days: []string{"2026-10-26"}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range edited {
		if s.SessionID == target.SessionID && s.Day == target.Day {
			found = true
			if !s.Edited {
				t.Fatal("segment should be marked as edited")
			}
			if !s.Start.Equal(start) || !s.End.Equal(end) {
				t.Fatalf("times = %v-%v, want %v-%v", s.Start, s.End, start, end)
			}
		}
	}
	if !found {
		t.Fatal("edited segment not found")
	}

	if err := st.ResetTimes(target.SessionID, target.Day); err != nil {
		t.Fatal(err)
	}
	reset, err := st.Segments(Filter{Days: []string{"2026-10-26"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range reset {
		if s.SessionID == target.SessionID && s.Edited {
			t.Fatal("override should have been removed")
		}
	}

	// Re-synchronising must preserve overrides.
	if err := st.UpdateTimes(target.SessionID, target.Day, start, end); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Sync(src); err != nil {
		t.Fatal(err)
	}
	after, err := st.Segments(Filter{Days: []string{"2026-10-26"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range after {
		if s.SessionID == target.SessionID && !s.Edited {
			t.Fatal("override lost after sync")
		}
	}
}

func createSourceV2(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE session_v2 (id TEXT PRIMARY KEY, directory TEXT NOT NULL, title TEXT, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL)`,
		`CREATE TABLE session_message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, type TEXT NOT NULL, seq INTEGER NOT NULL, time_created INTEGER NOT NULL, data TEXT NOT NULL)`,
		`CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT, title TEXT, time_created INTEGER, time_updated INTEGER)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, data TEXT)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	at := func(d, h, m int) int64 {
		return time.Date(2026, 11, d, h, m, 0, 0, time.Local).UnixMilli()
	}
	messages := []struct {
		id, session, kind string
		start, completed  int64
		tokIn, tokOut     int
	}{
		{"m1", "v1", "user", at(5, 9, 0), 0, 0, 0},
		{"m2", "v1", "assistant", at(5, 9, 0), at(5, 9, 10), 100, 20},
		{"m3", "v1", "assistant", at(5, 9, 20), at(5, 9, 30), 50, 10},
		{"m4", "v1", "idle", at(5, 11, 0), 0, 0, 0},
		{"m5", "v2", "assistant", at(6, 10, 0), at(6, 10, 5), 200, 40},
	}
	for i, m := range messages {
		data := fmt.Sprintf(`{"time":{"created":%d,"completed":%d},"cost":0.1,"tokens":{"input":%d,"output":%d,"reasoning":0,"cache":{"read":0,"write":0}}}`,
			m.start, m.completed, m.tokIn, m.tokOut)
		if _, err := db.Exec(`INSERT INTO session_message(id,session_id,type,seq,time_created,data) VALUES(?,?,?,?,?,?)`,
			m.id, m.session, m.kind, i+1, m.start, data); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO session_v2(id,directory,title,time_created,time_updated) VALUES(?,?,?,?,?)`,
		"v1", "/home/admin/Proyectos/alpha", "v2 title", at(5, 9, 0), at(5, 11, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session_v2(id,directory,title,time_created,time_updated) VALUES(?,?,?,?,?)`,
		"v2", "/home/admin/Proyectos/beta", nil, at(6, 10, 0), at(6, 10, 5)); err != nil {
		t.Fatal(err)
	}
	// Legacy-only session (partial migration) plus a duplicate already present in V2.
	if _, err := db.Exec(`INSERT INTO session(id,directory,title,time_created,time_updated) VALUES(?,?,?,?,?)`,
		"old", "/home/admin/Proyectos/gamma", "old title", at(1, 8, 0), at(1, 8, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session(id,directory,title,time_created,time_updated) VALUES(?,?,?,?,?)`,
		"v1", "/home/admin/Proyectos/alpha", "legacy duplicate", at(5, 9, 0), at(5, 9, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestSyncPrefersV2Schema(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	createSourceV2(t, src)
	st, err := New(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	stats, err := st.Sync(src)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sessions != 3 || stats.Segments != 3 {
		t.Fatalf("stats = %+v, want 3 sessions and 3 segments", stats)
	}

	segs, err := st.Segments(Filter{Days: []string{"2026-11-05"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 {
		t.Fatalf("segments = %d, want 1", len(segs))
	}
	seg := segs[0]
	if seg.Title != "v2 title" {
		t.Fatalf("title = %q, want v2 title", seg.Title)
	}
	if got := seg.End.Format("15:04"); got != "09:30" {
		t.Fatalf("end = %s, want 09:30 (idle events must be ignored)", got)
	}
	if seg.Tokens.Total() != 180 || seg.Cost < 0.19 || seg.Cost > 0.21 {
		t.Fatalf("tokens/cost = %d/%.2f, want 180/0.20", seg.Tokens.Total(), seg.Cost)
	}

	nov6, err := st.Segments(Filter{Days: []string{"2026-11-06"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(nov6) != 1 || nov6[0].Title != "" {
		t.Fatalf("nov6 = %+v, want one segment with empty title", nov6)
	}

	old, err := st.Segments(Filter{Days: []string{"2026-11-01"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 1 || old[0].SessionID != "old" {
		t.Fatalf("legacy-only session missing: %+v", old)
	}
}

func TestFilterProjectAndMonth(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "opencode.db")
	createSource(t, src)
	st, err := New(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.Sync(src); err != nil {
		t.Fatal(err)
	}
	segs, err := st.Segments(Filter{Month: "2026-10", Project: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 || segs[0].SessionID != "s1" {
		t.Fatalf("unexpected segments: %+v", segs)
	}
	if _, err := st.Segments(Filter{Days: []string{"2026-10-25"}}); err != nil {
		t.Fatal(err)
	}
}
