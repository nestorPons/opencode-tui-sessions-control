package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nestorpons/sessions-control/internal/config"
	"github.com/nestorpons/sessions-control/internal/export"
	"github.com/nestorpons/sessions-control/internal/report"
	"github.com/nestorpons/sessions-control/internal/store"
	"github.com/nestorpons/sessions-control/internal/tui"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		source      = flag.String("source", "", "ruta a opencode.db (por defecto ~/.local/share/opencode/opencode.db)")
		dbPath      = flag.String("db", "", "ruta a la base de datos local de sessions-control")
		exportsDir  = flag.String("exports", "", "directorio de exportaciones")
		noSync      = flag.Bool("no-sync", false, "no sincronizar con opencode.db al arrancar")
		viewFlag    = flag.String("view", "sessions", "vista: sessions, projects, months, days")
		daysFlag    = flag.String("days", "", "días separados por comas: 2026-10-26,2026-10-27")
		fromFlag    = flag.String("from", "", "día inicial YYYY-MM-DD")
		toFlag      = flag.String("to", "", "día final YYYY-MM-DD")
		monthFlag   = flag.String("month", "", "mes YYYY-MM")
		project     = flag.String("project", "", "proyecto (nombre o ruta)")
		exportFlag  = flag.String("export", "", "exportar: xlsx o pdf")
		outFlag     = flag.String("out", "", "ruta del archivo exportado")
		printFlag   = flag.Bool("print", false, "imprimir la tabla por stdout y salir")
		resetFlag   = flag.Bool("reset-overrides", false, "borrar todas las horas editadas manualmente")
		versionFlag = flag.Bool("version", false, "mostrar versión")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Println("sessions-control", version)
		return nil
	}

	cfg, err := config.Default()
	if err != nil {
		return err
	}
	if *source != "" {
		cfg.SourceDB = *source
	}
	if *dbPath != "" {
		cfg.AppDB = *dbPath
	}
	if *exportsDir != "" {
		cfg.ExportsDir = *exportsDir
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	st, err := store.New(cfg.AppDB)
	if err != nil {
		return err
	}
	defer st.Close()

	interactive := !*printFlag && *exportFlag == ""

	if !*noSync {
		stats, err := st.Sync(cfg.SourceDB)
		if err != nil {
			if interactive {
				fmt.Fprintln(os.Stderr, "aviso: no se pudo sincronizar:", err)
			} else {
				return err
			}
		} else if !interactive {
			fmt.Fprintf(os.Stderr, "sincronizado: %d sesiones, %d tramos desde %s\n",
				stats.Sessions, stats.Segments, stats.Source)
		}
	}
	if *resetFlag {
		if err := st.ResetAll(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "horas manuales eliminadas")
	}

	stats, err := st.Stats()
	if err != nil {
		return err
	}

	if *printFlag || *exportFlag != "" {
		view, ok := report.ParseView(*viewFlag)
		if !ok {
			return fmt.Errorf("vista no válida %q (sessions|projects|months|days)", *viewFlag)
		}
		filter, err := buildFilter(*daysFlag, *fromFlag, *toFlag, *monthFlag, *project)
		if err != nil {
			return err
		}
		segs, err := st.Segments(filter)
		if err != nil {
			return err
		}
		rep := report.Build(view, segs, filter.Describe())

		if *printFlag {
			printReport(rep)
			return nil
		}

		format := strings.ToLower(*exportFlag)
		if format != "xlsx" && format != "pdf" {
			return fmt.Errorf("formato no válido %q (xlsx|pdf)", *exportFlag)
		}
		out := *outFlag
		if out == "" {
			ext := format
			out = cfg.ExportsDir + "/sessions-control_" + string(view) + "_" + time.Now().Format("20060102-150405") + "." + ext
		}
		if format == "pdf" {
			err = export.PDF(out, rep, stats)
		} else {
			err = export.Excel(out, rep, stats)
		}
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	model := tui.New(st, cfg, stats)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func buildFilter(days, from, to, month, project string) (store.Filter, error) {
	f := store.Filter{From: from, To: to, Month: month, Project: project}
	for _, d := range strings.Split(days, ",") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return f, fmt.Errorf("día no válido %q (usa YYYY-MM-DD)", d)
		}
		f.Days = append(f.Days, d)
	}
	if from != "" {
		if _, err := time.Parse("2006-01-02", from); err != nil {
			return f, fmt.Errorf("from no válido %q (usa YYYY-MM-DD)", from)
		}
	}
	if to != "" {
		if _, err := time.Parse("2006-01-02", to); err != nil {
			return f, fmt.Errorf("to no válido %q (usa YYYY-MM-DD)", to)
		}
	}
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			return f, fmt.Errorf("month no válido %q (usa YYYY-MM)", month)
		}
	}
	return f, nil
}

func printReport(r report.Report) {
	fmt.Println(r.Title)
	fmt.Println("Filtro:", r.Filter)
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(r.Headers, "\t"))
	for _, row := range r.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	fmt.Fprintln(w, strings.Join(r.Totals, "\t"))
	w.Flush()
	fmt.Println(r.Summary)
}
