package report

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nestorpons/sessions-control/internal/domain"
)

type View string

const (
	ViewSessions View = "sessions"
	ViewProjects View = "projects"
	ViewMonths   View = "months"
	ViewDays     View = "days"
)

func ParseView(s string) (View, bool) {
	switch View(s) {
	case ViewSessions:
		return ViewSessions, true
	case ViewProjects:
		return ViewProjects, true
	case ViewMonths:
		return ViewMonths, true
	case ViewDays:
		return ViewDays, true
	}
	return ViewSessions, false
}

type Report struct {
	View      View
	Title     string
	Filter    string
	Headers   []string
	Rows      [][]string
	Data      [][]any
	Totals    []string
	TotalData []any
	Summary   string
	Raw       []domain.Segment
}

func Build(view View, segs []domain.Segment, filterDesc string, gap, dayGap time.Duration, rate float64) Report {
	r := Report{View: view, Filter: filterDesc, Raw: segs}
	switch view {
	case ViewProjects:
		groups := domain.AggregateByProject(segs)
		r.Title = "Totales por proyecto"
		r.Headers = []string{"Proyecto", "Fact.", "Inicio", "Fin", "Sesiones", "Días", "Horas", "Efectivo", "Reparto", "Importe", "Único", "Tokens", "Coste"}
		for _, g := range groups {
			r.Rows = append(r.Rows, []string{
				g.Label, fact(g.Internal), clock(g.Start), clock(g.End), itoa(g.Sessions), itoa(g.Days),
				hours(g.Worked), hours(g.Effective), hours(g.Allocated), euros(billing(g.Allocated, rate)),
				hours(g.Unique), tokens(g.Tokens.Total()), money(g.Cost),
			})
			r.Data = append(r.Data, []any{
				g.Label, fact(g.Internal), clock(g.Start), clock(g.End), g.Sessions, g.Days,
				hoursFloat(g.Worked), hoursFloat(g.Effective), hoursFloat(g.Allocated), billing(g.Allocated, rate),
				hoursFloat(g.Unique), g.Tokens.Total(), round(g.Cost),
			})
		}
		r.Totals = []string{"TOTAL", "", "", "", itoa(countSessions(segs)), itoa(domain.DistinctDays(segs)),
			hours(domain.SumDurations(segs)), hours(domain.SumEffective(segs)), hours(domain.SumAllocated(segs)),
			euros(billing(domain.SumAllocated(segs), rate)), hours(domain.UniqueDuration(segs)),
			tokens(sumTokens(segs).Total()), money(sumCost(segs))}
		r.TotalData = []any{"TOTAL", "", "", "", countSessions(segs), domain.DistinctDays(segs),
			hoursFloat(domain.SumDurations(segs)), hoursFloat(domain.SumEffective(segs)), hoursFloat(domain.SumAllocated(segs)),
			billing(domain.SumAllocated(segs), rate), hoursFloat(domain.UniqueDuration(segs)),
			sumTokens(segs).Total(), round(sumCost(segs))}
		r.Summary = fmt.Sprintf("%s · %s · %s trabajadas · %s efectivas · %s repartidas · %s · %s únicas",
			plural(len(groups), "proyecto", "proyectos"), plural(countSessions(segs), "sesión", "sesiones"),
			humanDuration(domain.SumDurations(segs)), humanDuration(domain.SumEffective(segs)),
			humanDuration(domain.SumAllocated(segs)), euros(billing(domain.SumAllocated(segs), rate)),
			humanDuration(domain.UniqueDuration(segs))) + rulesNote(gap, dayGap)
	case ViewMonths:
		groups := domain.AggregateByMonth(segs)
		r.Title = "Totales por mes"
		r.Headers = []string{"Mes", "Sesiones", "Días", "Horas", "Efectivo", "Reparto", "Importe", "Único", "Tokens", "Coste"}
		for _, g := range groups {
			r.Rows = append(r.Rows, []string{
				g.Label, itoa(g.Sessions), itoa(g.Days), hours(g.Worked), hours(g.Effective),
				hours(g.Allocated), euros(billing(g.Allocated, rate)), hours(g.Unique),
				tokens(g.Tokens.Total()), money(g.Cost),
			})
			r.Data = append(r.Data, []any{
				g.Label, g.Sessions, g.Days, hoursFloat(g.Worked), hoursFloat(g.Effective),
				hoursFloat(g.Allocated), billing(g.Allocated, rate), hoursFloat(g.Unique),
				g.Tokens.Total(), round(g.Cost),
			})
		}
		r.Totals = []string{"TOTAL", itoa(countSessions(segs)), itoa(domain.DistinctDays(segs)),
			hours(domain.SumDurations(segs)), hours(domain.SumEffective(segs)), hours(domain.SumAllocated(segs)),
			euros(billing(domain.SumAllocated(segs), rate)), hours(domain.UniqueDuration(segs)),
			tokens(sumTokens(segs).Total()), money(sumCost(segs))}
		r.TotalData = []any{"TOTAL", countSessions(segs), domain.DistinctDays(segs),
			hoursFloat(domain.SumDurations(segs)), hoursFloat(domain.SumEffective(segs)), hoursFloat(domain.SumAllocated(segs)),
			billing(domain.SumAllocated(segs), rate), hoursFloat(domain.UniqueDuration(segs)),
			sumTokens(segs).Total(), round(sumCost(segs))}
		r.Summary = fmt.Sprintf("%s · %s trabajadas · %s efectivas · %s repartidas · %s · %s únicas",
			plural(len(groups), "mes", "meses"),
			humanDuration(domain.SumDurations(segs)), humanDuration(domain.SumEffective(segs)),
			humanDuration(domain.SumAllocated(segs)), euros(billing(domain.SumAllocated(segs), rate)),
			humanDuration(domain.UniqueDuration(segs))) + rulesNote(gap, dayGap)
	case ViewDays:
		groups := domain.AggregateByDay(segs)
		r.Title = "Totales por día"
		r.Headers = []string{"Fecha", "Sesiones", "Horas", "Efectivo", "Reparto", "Importe", "Único", "Tokens", "Coste"}
		for _, g := range groups {
			r.Rows = append(r.Rows, []string{
				g.Label, itoa(g.Sessions), hours(g.Worked), hours(g.Effective), hours(g.Allocated),
				euros(billing(g.Allocated, rate)), hours(g.Unique), tokens(g.Tokens.Total()), money(g.Cost),
			})
			r.Data = append(r.Data, []any{
				g.Label, g.Sessions, hoursFloat(g.Worked), hoursFloat(g.Effective), hoursFloat(g.Allocated),
				billing(g.Allocated, rate), hoursFloat(g.Unique), g.Tokens.Total(), round(g.Cost),
			})
		}
		r.Totals = []string{"TOTAL", itoa(countSessions(segs)), hours(domain.SumDurations(segs)),
			hours(domain.SumEffective(segs)), hours(domain.SumAllocated(segs)),
			euros(billing(domain.SumAllocated(segs), rate)), hours(domain.UniqueDuration(segs)),
			tokens(sumTokens(segs).Total()), money(sumCost(segs))}
		r.TotalData = []any{"TOTAL", countSessions(segs), hoursFloat(domain.SumDurations(segs)),
			hoursFloat(domain.SumEffective(segs)), hoursFloat(domain.SumAllocated(segs)),
			billing(domain.SumAllocated(segs), rate), hoursFloat(domain.UniqueDuration(segs)),
			sumTokens(segs).Total(), round(sumCost(segs))}
		r.Summary = fmt.Sprintf("%s · %s trabajadas · %s efectivas · %s repartidas · %s · %s únicas",
			plural(len(groups), "día", "días"),
			humanDuration(domain.SumDurations(segs)), humanDuration(domain.SumEffective(segs)),
			humanDuration(domain.SumAllocated(segs)), euros(billing(domain.SumAllocated(segs), rate)),
			humanDuration(domain.UniqueDuration(segs))) + rulesNote(gap, dayGap)
	default:
		r.Title = "Sesiones (una fila por sesión y día)"
		r.Headers = []string{"Fecha", "Inicio", "Fin", "Horas", "Efectivo", "Reparto", "Importe", "Proyecto", "Sesión", "Tokens", "Coste"}
		for _, s := range segs {
			start, end := clock(s.Start), clock(s.End)
			if s.Edited {
				start += "*"
				end += "*"
			}
			r.Rows = append(r.Rows, []string{
				s.Day, start, end, hours(s.Duration()), hours(s.Effective), hours(s.Allocated),
				euros(billing(s.Allocated, rate)), s.ProjectName, s.Title, tokens(s.Tokens.Total()), money(s.Cost),
			})
			r.Data = append(r.Data, []any{
				s.Day, start, end, hoursFloat(s.Duration()), hoursFloat(s.Effective), hoursFloat(s.Allocated),
				billing(s.Allocated, rate), s.ProjectName, s.Title, s.Tokens.Total(), round(s.Cost),
			})
		}
		r.Totals = []string{"TOTAL", "", "", hours(domain.SumDurations(segs)), hours(domain.SumEffective(segs)),
			hours(domain.SumAllocated(segs)), euros(billing(domain.SumAllocated(segs), rate)), "", "",
			tokens(sumTokens(segs).Total()), money(sumCost(segs))}
		r.TotalData = []any{"TOTAL", "", "", hoursFloat(domain.SumDurations(segs)), hoursFloat(domain.SumEffective(segs)),
			hoursFloat(domain.SumAllocated(segs)), billing(domain.SumAllocated(segs), rate), "", "",
			sumTokens(segs).Total(), round(sumCost(segs))}
		r.Summary = fmt.Sprintf("%s · %s · %s trabajadas · %s efectivas · %s repartidas · %s · %s únicas",
			plural(countSessions(segs), "sesión", "sesiones"), plural(len(segs), "tramo", "tramos"),
			humanDuration(domain.SumDurations(segs)), humanDuration(domain.SumEffective(segs)),
			humanDuration(domain.SumAllocated(segs)), euros(billing(domain.SumAllocated(segs), rate)),
			humanDuration(domain.UniqueDuration(segs))) + rulesNote(gap, dayGap)
	}
	return r
}

// rulesNote explains the effective and allocation rules at the end of the summary.
func rulesNote(gap, dayGap time.Duration) string {
	var parts []string
	if gap > 0 {
		parts = append(parts, "efectivo ≤ "+humanDuration(gap))
	}
	if dayGap > 0 {
		parts = append(parts, "reparto ≤ "+humanDuration(dayGap))
	}
	if len(parts) == 0 {
		return ""
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return " · " + out
}

func fact(internal bool) string {
	if internal {
		return "no"
	}
	return "sí"
}

func clock(t time.Time) string {
	return t.Format("15:04")
}

func hours(d time.Duration) string { return fmt.Sprintf("%.2f", hoursFloat(d)) }

func hoursFloat(d time.Duration) float64 { return float64(d) / float64(time.Hour) }

func tokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.2f M", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1f k", float64(n)/1_000)
	default:
		return strconv.FormatInt(n, 10)
	}
}

func money(f float64) string { return fmt.Sprintf("%.4f", f) }

// billing converts an allocated duration to euros at the hourly rate.
func billing(d time.Duration, rate float64) float64 {
	return float64(int64(hoursFloat(d)*rate*100+0.5)) / 100
}

// euros renders an amount with a Spanish decimal comma.
func euros(v float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", v), ".", ",") + " €"
}

func round(f float64) float64 { return float64(int64(f*10000+0.5)) / 10000 }

func itoa(n int) string { return strconv.Itoa(n) }

func sumTokens(segs []domain.Segment) domain.Tokens {
	var t domain.Tokens
	for _, s := range segs {
		t = t.Add(s.Tokens)
	}
	return t
}

func sumCost(segs []domain.Segment) float64 {
	var f float64
	for _, s := range segs {
		f += s.Cost
	}
	return f
}

func countSessions(segs []domain.Segment) int {
	seen := map[string]struct{}{}
	for _, s := range segs {
		seen[s.SessionID] = struct{}{}
	}
	return len(seen)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func humanDuration(d time.Duration) string {
	total := int64(d) / int64(time.Minute)
	h := total / 60
	m := total % 60
	if h == 0 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %02d min", h, m)
}
