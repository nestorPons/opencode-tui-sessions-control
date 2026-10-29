package domain

import (
	"path/filepath"
	"sort"
	"time"
)

// Tokens is the token usage of one or more assistant messages.
type Tokens struct {
	Input      int64
	Output     int64
	Reasoning  int64
	CacheRead  int64
	CacheWrite int64
}

func (t Tokens) Total() int64 {
	return t.Input + t.Output + t.Reasoning + t.CacheRead + t.CacheWrite
}

func (t Tokens) Add(o Tokens) Tokens {
	return Tokens{
		Input:      t.Input + o.Input,
		Output:     t.Output + o.Output,
		Reasoning:  t.Reasoning + o.Reasoning,
		CacheRead:  t.CacheRead + o.CacheRead,
		CacheWrite: t.CacheWrite + o.CacheWrite,
	}
}

// Segment is the activity of one session during one local day.
type Segment struct {
	SessionID   string
	Day         string // YYYY-MM-DD (local)
	Project     string // absolute directory
	ProjectName string
	Title       string
	Start       time.Time
	End         time.Time
	Edited      bool
	Tokens      Tokens
	Cost        float64
}

func (s Segment) Duration() time.Duration { return s.End.Sub(s.Start) }

// Session aggregates every visible segment of the same session.
type Session struct {
	ID          string
	Project     string
	ProjectName string
	Title       string
	Start       time.Time
	End         time.Time
	Days        int
	Worked      time.Duration // sum of segments
	Unique      time.Duration // union of segments
	Tokens      Tokens
	Cost        float64
	Edited      bool
}

// Group is an aggregated set of segments (project, month or day).
type Group struct {
	Key      string
	Label    string
	Sessions int
	Days     int
	Start    time.Time
	End      time.Time
	Worked   time.Duration // sum of durations, overlaps counted
	Unique   time.Duration // union of intervals
	Tokens   Tokens
	Cost     float64
}

func ProjectName(dir string) string {
	if dir == "" {
		return "(sin proyecto)"
	}
	clean := filepath.Clean(dir)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) {
		return clean
	}
	// ~/.local/... sessions run inside the home directory: label them "global".
	if clean == "/home/"+base {
		return "global"
	}
	return base
}

// UniqueDuration returns the union of all intervals.
func UniqueDuration(segs []Segment) time.Duration {
	if len(segs) == 0 {
		return 0
	}
	sorted := append([]Segment(nil), segs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })
	var total time.Duration
	curStart, curEnd := sorted[0].Start, sorted[0].End
	for _, s := range sorted[1:] {
		if !s.Start.After(curEnd) {
			if s.End.After(curEnd) {
				curEnd = s.End
			}
			continue
		}
		total += curEnd.Sub(curStart)
		curStart, curEnd = s.Start, s.End
	}
	return total + curEnd.Sub(curStart)
}

// SumDurations adds every segment duration.
func SumDurations(segs []Segment) time.Duration {
	var d time.Duration
	for _, s := range segs {
		d += s.Duration()
	}
	return d
}

// DistinctDays counts different calendar days.
func DistinctDays(segs []Segment) int {
	seen := map[string]struct{}{}
	for _, s := range segs {
		seen[s.Day] = struct{}{}
	}
	return len(seen)
}

// AggregateSessions groups segments by session (descending by start).
func AggregateSessions(segs []Segment) []Session {
	index := map[string]int{}
	var out []Session
	for _, s := range segs {
		i, ok := index[s.SessionID]
		if !ok {
			out = append(out, Session{
				ID:          s.SessionID,
				Project:     s.Project,
				ProjectName: s.ProjectName,
				Title:       s.Title,
				Start:       s.Start,
				End:         s.End,
			})
			i = len(out) - 1
			index[s.SessionID] = i
		}
		agg := &out[i]
		if s.Start.Before(agg.Start) {
			agg.Start = s.Start
		}
		if s.End.After(agg.End) {
			agg.End = s.End
		}
		agg.Days++
		agg.Worked += s.Duration()
		agg.Tokens = agg.Tokens.Add(s.Tokens)
		agg.Cost += s.Cost
		agg.Edited = agg.Edited || s.Edited
	}
	for i := range out {
		sub := segmentsOf(segs, out[i].ID)
		out[i].Unique = UniqueDuration(sub)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Start.Equal(out[j].Start) {
			return out[i].ID > out[j].ID
		}
		return out[i].Start.After(out[j].Start)
	})
	return out
}

func segmentsOf(segs []Segment, sessionID string) []Segment {
	var out []Segment
	for _, s := range segs {
		if s.SessionID == sessionID {
			out = append(out, s)
		}
	}
	return out
}

// AggregateByProject groups segments by project, ordered by unique time.
func AggregateByProject(segs []Segment) []Group {
	index := map[string]int{}
	var out []Group
	for _, s := range segs {
		i, ok := index[s.Project]
		if !ok {
			out = append(out, Group{Key: s.Project, Label: s.ProjectName})
			i = len(out) - 1
			index[s.Project] = i
		}
		g := &out[i]
		g.add(s)
	}
	for i := range out {
		var sub []Segment
		for _, s := range segs {
			if s.Project == out[i].Key {
				sub = append(sub, s)
			}
		}
		out[i].Unique = UniqueDuration(sub)
		out[i].Sessions = countSessions(sub)
		out[i].Days = DistinctDays(sub)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Unique == out[j].Unique {
			return out[i].Label < out[j].Label
		}
		return out[i].Unique > out[j].Unique
	})
	return out
}

// AggregateByMonth groups segments by YYYY-MM, ordered descending.
func AggregateByMonth(segs []Segment) []Group {
	index := map[string]int{}
	var out []Group
	for _, s := range segs {
		key := s.Day
		if len(key) >= 7 {
			key = key[:7]
		}
		i, ok := index[key]
		if !ok {
			out = append(out, Group{Key: key, Label: key})
			i = len(out) - 1
			index[key] = i
		}
		out[i].add(s)
	}
	for i := range out {
		var sub []Segment
		for _, s := range segs {
			key := s.Day
			if len(key) >= 7 {
				key = key[:7]
			}
			if key == out[i].Key {
				sub = append(sub, s)
			}
		}
		out[i].Unique = UniqueDuration(sub)
		out[i].Sessions = countSessions(sub)
		out[i].Days = DistinctDays(sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key > out[j].Key })
	return out
}

// AggregateByDay groups segments by day, ordered descending.
func AggregateByDay(segs []Segment) []Group {
	index := map[string]int{}
	var out []Group
	for _, s := range segs {
		i, ok := index[s.Day]
		if !ok {
			out = append(out, Group{Key: s.Day, Label: s.Day})
			i = len(out) - 1
			index[s.Day] = i
		}
		out[i].add(s)
	}
	for i := range out {
		var sub []Segment
		for _, s := range segs {
			if s.Day == out[i].Key {
				sub = append(sub, s)
			}
		}
		out[i].Unique = UniqueDuration(sub)
		out[i].Sessions = countSessions(sub)
		out[i].Days = DistinctDays(sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key > out[j].Key })
	return out
}

func (g *Group) add(s Segment) {
	if g.Start.IsZero() || s.Start.Before(g.Start) {
		g.Start = s.Start
	}
	if g.End.IsZero() || s.End.After(g.End) {
		g.End = s.End
	}
	g.Worked += s.Duration()
	g.Tokens = g.Tokens.Add(s.Tokens)
	g.Cost += s.Cost
}

func countSessions(segs []Segment) int {
	seen := map[string]struct{}{}
	for _, s := range segs {
		seen[s.SessionID] = struct{}{}
	}
	return len(seen)
}
