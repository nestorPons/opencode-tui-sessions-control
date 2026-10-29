package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/nestorpons/sessions-control/internal/domain"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS session_day (
	session_id         TEXT NOT NULL,
	day                TEXT NOT NULL,
	project            TEXT NOT NULL,
	project_name       TEXT NOT NULL,
	title              TEXT NOT NULL,
	start_ms           INTEGER NOT NULL,
	end_ms             INTEGER NOT NULL,
	tokens_input       INTEGER NOT NULL DEFAULT 0,
	tokens_output      INTEGER NOT NULL DEFAULT 0,
	tokens_reasoning   INTEGER NOT NULL DEFAULT 0,
	tokens_cache_read  INTEGER NOT NULL DEFAULT 0,
	tokens_cache_write INTEGER NOT NULL DEFAULT 0,
	cost               REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (session_id, day)
);
CREATE INDEX IF NOT EXISTS session_day_day_idx ON session_day(day);
CREATE INDEX IF NOT EXISTS session_day_project_idx ON session_day(project);
CREATE TABLE IF NOT EXISTS time_override (
	session_id TEXT NOT NULL,
	day        TEXT NOT NULL,
	start_ms   INTEGER NOT NULL,
	end_ms     INTEGER NOT NULL,
	updated_ms INTEGER NOT NULL,
	PRIMARY KEY (session_id, day)
);
CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

type Store struct {
	db *sql.DB
}

type Stats struct {
	Source      string
	SyncedAt    time.Time
	Sessions    int
	Segments    int
	Overrides   int
	SourceFound bool
}

type ProjectRef struct {
	Path string
	Name string
}

// Filter selects day-level segments.
type Filter struct {
	Days    []string // exact YYYY-MM-DD values
	From    string   // inclusive YYYY-MM-DD
	To      string   // inclusive YYYY-MM-DD
	Month   string   // YYYY-MM
	Project string   // path or name
}

func (f Filter) IsEmpty() bool {
	return len(f.Days) == 0 && f.From == "" && f.To == "" && f.Month == "" && f.Project == ""
}

func (f Filter) Describe() string {
	var parts []string
	switch {
	case len(f.Days) > 0:
		parts = append(parts, "días="+strings.Join(f.Days, ","))
	case f.From != "" || f.To != "":
		parts = append(parts, "rango="+orAll(f.From)+".."+orAll(f.To))
	}
	if f.Month != "" {
		parts = append(parts, "mes="+f.Month)
	}
	if f.Project != "" {
		parts = append(parts, "proyecto="+f.Project)
	}
	if len(parts) == 0 {
		return "todo"
	}
	return strings.Join(parts, " · ")
}

func orAll(s string) string {
	if s == "" {
		return "*"
	}
	return s
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type rawMessage struct {
	Role   string  `json:"role"`
	Cost   float64 `json:"cost"`
	Tokens struct {
		Input     int64 `json:"input"`
		Output    int64 `json:"output"`
		Reasoning int64 `json:"reasoning"`
		Cache     struct {
			Read  int64 `json:"read"`
			Write int64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens"`
	Time struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
}

type dayBucket struct {
	start  int64
	end    int64
	tokens domain.Tokens
	cost   float64
}

// Sync rebuilds session_day from an opencode SQLite database.
func (s *Store) Sync(source string) (Stats, error) {
	src, err := os.Stat(source)
	if err != nil {
		return Stats{}, fmt.Errorf("source database %q: %w", source, err)
	}
	if src.IsDir() {
		return Stats{}, fmt.Errorf("source database %q is a directory", source)
	}
	srcDB, err := sql.Open("sqlite", "file:"+source+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return Stats{}, fmt.Errorf("open source: %w", err)
	}
	defer srcDB.Close()
	if err := srcDB.Ping(); err != nil {
		return Stats{}, fmt.Errorf("ping source: %w", err)
	}

	rows, err := srcDB.Query(`
		SELECT s.id, s.directory, s.title, s.time_created, s.time_updated,
		       m.time_created, m.data
		FROM session s
		LEFT JOIN message m ON m.session_id = s.id
		ORDER BY s.id, m.time_created, m.id`)
	if err != nil {
		return Stats{}, fmt.Errorf("query source: %w", err)
	}
	defer rows.Close()

	type sessionInfo struct {
		dir, title   string
		created      int64
		updated      int64
		days         map[string]*dayBucket
		messageCount int
	}
	sessions := map[string]*sessionInfo{}
	var order []string

	for rows.Next() {
		var (
			id, dir, title   string
			created, updated int64
			msgCreated       sql.NullInt64
			data             sql.NullString
		)
		if err := rows.Scan(&id, &dir, &title, &created, &updated, &msgCreated, &data); err != nil {
			return Stats{}, fmt.Errorf("scan source: %w", err)
		}
		info, ok := sessions[id]
		if !ok {
			info = &sessionInfo{dir: dir, title: title, created: created, updated: updated, days: map[string]*dayBucket{}}
			sessions[id] = info
			order = append(order, id)
		}
		if !msgCreated.Valid {
			continue
		}
		info.messageCount++
		start := msgCreated.Int64
		end := start
		var tok domain.Tokens
		var cost float64
		if data.Valid {
			var msg rawMessage
			if err := json.Unmarshal([]byte(data.String), &msg); err == nil {
				if msg.Time.Completed > 0 {
					end = msg.Time.Completed
				}
				if msg.Time.Created > 0 {
					start = msg.Time.Created
				}
				if msg.Role == "assistant" {
					tok = domain.Tokens{
						Input:      msg.Tokens.Input,
						Output:     msg.Tokens.Output,
						Reasoning:  msg.Tokens.Reasoning,
						CacheRead:  msg.Tokens.Cache.Read,
						CacheWrite: msg.Tokens.Cache.Write,
					}
					cost = msg.Cost
				}
			}
		}
		if end < start {
			end = start
		}
		day := time.UnixMilli(start).Local().Format("2006-01-02")
		bucket, ok := info.days[day]
		if !ok {
			bucket = &dayBucket{start: start, end: end}
			info.days[day] = bucket
		}
		if start < bucket.start {
			bucket.start = start
		}
		if end > bucket.end {
			bucket.end = end
		}
		bucket.tokens = bucket.tokens.Add(tok)
		bucket.cost += cost
	}
	if err := rows.Err(); err != nil {
		return Stats{}, fmt.Errorf("iterate source: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM session_day`); err != nil {
		return Stats{}, fmt.Errorf("clear session_day: %w", err)
	}
	insert, err := tx.Prepare(`INSERT INTO session_day
		(session_id, day, project, project_name, title, start_ms, end_ms,
		 tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_cache_write, cost)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return Stats{}, err
	}
	defer insert.Close()

	segments := 0
	for _, id := range order {
		info := sessions[id]
		if info.messageCount == 0 {
			day := time.UnixMilli(info.created).Local().Format("2006-01-02")
			info.days[day] = &dayBucket{start: info.created, end: info.updated}
		}
		days := make([]string, 0, len(info.days))
		for d := range info.days {
			days = append(days, d)
		}
		sort.Strings(days)
		name := domain.ProjectName(info.dir)
		for _, d := range days {
			b := info.days[d]
			if _, err := insert.Exec(id, d, info.dir, name, info.title, b.start, b.end,
				b.tokens.Input, b.tokens.Output, b.tokens.Reasoning,
				b.tokens.CacheRead, b.tokens.CacheWrite, b.cost); err != nil {
				return Stats{}, fmt.Errorf("insert segment: %w", err)
			}
			segments++
		}
	}

	now := time.Now()
	setMeta(tx, "source", source)
	setMeta(tx, "synced_at", now.Format(time.RFC3339))
	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}
	return Stats{
		Source:      source,
		SyncedAt:    now,
		Sessions:    len(sessions),
		Segments:    segments,
		SourceFound: true,
	}, nil
}

func setMeta(tx *sql.Tx, key, value string) {
	_, _ = tx.Exec(`INSERT INTO meta(key,value) VALUES(?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
}

// Segments returns the segments matching f, with manual times applied.
func (s *Store) Segments(f Filter) ([]domain.Segment, error) {
	where, args := f.clause()
	query := `
		SELECT sd.session_id, sd.day, sd.project, sd.project_name, sd.title,
		       COALESCE(o.start_ms, sd.start_ms), COALESCE(o.end_ms, sd.end_ms),
		       CASE WHEN o.session_id IS NULL THEN 0 ELSE 1 END,
		       sd.tokens_input, sd.tokens_output, sd.tokens_reasoning,
		       sd.tokens_cache_read, sd.tokens_cache_write, sd.cost
		FROM session_day sd
		LEFT JOIN time_override o ON o.session_id = sd.session_id AND o.day = sd.day
		` + where + `
		ORDER BY COALESCE(o.start_ms, sd.start_ms)`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Segment
	for rows.Next() {
		var (
			seg            domain.Segment
			startMs, endMs int64
			edited         int
		)
		if err := rows.Scan(&seg.SessionID, &seg.Day, &seg.Project, &seg.ProjectName, &seg.Title,
			&startMs, &endMs, &edited, &seg.Tokens.Input, &seg.Tokens.Output, &seg.Tokens.Reasoning,
			&seg.Tokens.CacheRead, &seg.Tokens.CacheWrite, &seg.Cost); err != nil {
			return nil, err
		}
		seg.Start = time.UnixMilli(startMs).Local()
		seg.End = time.UnixMilli(endMs).Local()
		seg.Edited = edited == 1
		out = append(out, seg)
	}
	return out, rows.Err()
}

func (f Filter) clause() (string, []any) {
	var conds []string
	var args []any
	if len(f.Days) > 0 {
		conds = append(conds, "sd.day IN ("+placeholders(len(f.Days))+")")
		for _, d := range f.Days {
			args = append(args, d)
		}
	} else {
		if f.From != "" {
			conds = append(conds, "sd.day >= ?")
			args = append(args, f.From)
		}
		if f.To != "" {
			conds = append(conds, "sd.day <= ?")
			args = append(args, f.To)
		}
	}
	if f.Month != "" {
		conds = append(conds, "substr(sd.day,1,7) = ?")
		args = append(args, f.Month)
	}
	if f.Project != "" {
		conds = append(conds, "(sd.project = ? OR sd.project_name = ?)")
		args = append(args, f.Project, f.Project)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (s *Store) UpdateTimes(sessionID, day string, start, end time.Time) error {
	if !end.After(start) {
		return errors.New("la hora de fin debe ser posterior a la de inicio")
	}
	_, err := s.db.Exec(`INSERT INTO time_override(session_id, day, start_ms, end_ms, updated_ms)
		VALUES(?,?,?,?,?)
		ON CONFLICT(session_id, day) DO UPDATE SET
			start_ms=excluded.start_ms, end_ms=excluded.end_ms, updated_ms=excluded.updated_ms`,
		sessionID, day, start.UnixMilli(), end.UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		return fmt.Errorf("save times: %w", err)
	}
	return nil
}

func (s *Store) ResetTimes(sessionID, day string) error {
	_, err := s.db.Exec(`DELETE FROM time_override WHERE session_id=? AND day=?`, sessionID, day)
	return err
}

func (s *Store) Days() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT day FROM session_day ORDER BY day DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) Months() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT substr(day,1,7) FROM session_day ORDER BY 1 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) Projects() ([]ProjectRef, error) {
	rows, err := s.db.Query(`SELECT project, project_name, COUNT(*) c
		FROM session_day GROUP BY project, project_name ORDER BY project_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProjectRef
	for rows.Next() {
		var p ProjectRef
		var c int
		if err := rows.Scan(&p.Path, &p.Name, &c); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	_ = s.db.QueryRow(`SELECT value FROM meta WHERE key='source'`).Scan(&st.Source)
	var synced string
	_ = s.db.QueryRow(`SELECT value FROM meta WHERE key='synced_at'`).Scan(&synced)
	if synced != "" {
		st.SyncedAt, _ = time.Parse(time.RFC3339, synced)
	}
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM session_day`).Scan(&st.Segments)
	_ = s.db.QueryRow(`SELECT COUNT(DISTINCT session_id) FROM session_day`).Scan(&st.Sessions)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM time_override`).Scan(&st.Overrides)
	if st.Source != "" {
		if _, err := os.Stat(st.Source); err == nil {
			st.SourceFound = true
		}
	}
	return st, nil
}

// InputString renders the filter in the compact TUI syntax.
func (f Filter) InputString() string {
	var parts []string
	parts = append(parts, f.Days...)
	if f.From != "" {
		parts = append(parts, "from="+f.From)
	}
	if f.To != "" {
		parts = append(parts, "to="+f.To)
	}
	if f.Month != "" {
		parts = append(parts, "month="+f.Month)
	}
	if f.Project != "" {
		parts = append(parts, "project="+f.Project)
	}
	return strings.Join(parts, " ")
}

func (s *Store) ResetAll() error {
	_, err := s.db.Exec(`DELETE FROM time_override`)
	return err
}
