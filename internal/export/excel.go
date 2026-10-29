package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/nestorpons/sessions-control/internal/domain"
	"github.com/nestorpons/sessions-control/internal/report"
	"github.com/nestorpons/sessions-control/internal/store"
	"github.com/xuri/excelize/v2"
)

func Excel(path string, r report.Report, st store.Stats) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := sheetName(r.View)
	f.SetSheetName("Sheet1", sheet)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2D3E50"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    thinBorder(),
	})
	if err != nil {
		return err
	}
	totalStyle, err := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"E8EDF2"}, Pattern: 1},
		Border: thinBorder(),
	})
	if err != nil {
		return err
	}
	cellStyle, err := f.NewStyle(&excelize.Style{Border: thinBorder()})
	if err != nil {
		return err
	}
	hourStyle, err := f.NewStyle(&excelize.Style{NumFmt: 2, Border: thinBorder()})
	if err != nil {
		return err
	}
	customMoney := "0.0000"
	moneyStyle, err := f.NewStyle(&excelize.Style{NumFmt: 164, CustomNumFmt: &customMoney, Border: thinBorder()})
	if err != nil {
		return err
	}

	cols := len(r.Headers)
	lastCol, _ := excelize.ColumnNumberToName(cols)
	_ = f.MergeCell(sheet, "A1", lastCol+"1")
	_ = f.SetCellValue(sheet, "A1", r.Title)
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 16}})
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 24)

	meta := [][2]string{
		{"Generado", time.Now().Format("2006-01-02 15:04:05")},
		{"Filtro", r.Filter},
		{"Fuente", st.Source},
		{"Resumen", r.Summary},
	}
	for i, kv := range meta {
		row := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), kv[0])
		_ = f.MergeCell(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%s%d", lastCol, row))
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), kv[1])
	}

	headerRow := 7
	for i, h := range r.Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(sheet, cell, h)
	}
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", headerRow), fmt.Sprintf("%s%d", lastCol, headerRow), headerStyle)

	for ri, row := range r.Data {
		rowNum := headerRow + 1 + ri
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, rowNum)
			_ = f.SetCellValue(sheet, cell, val)
			switch val.(type) {
			case float64:
				if isMoneyColumn(r.Headers[ci]) {
					_ = f.SetCellStyle(sheet, cell, cell, moneyStyle)
				} else {
					_ = f.SetCellStyle(sheet, cell, cell, hourStyle)
				}
			default:
				_ = f.SetCellStyle(sheet, cell, cell, cellStyle)
			}
		}
	}

	totalRow := headerRow + len(r.Data) + 1
	for ci, val := range r.TotalData {
		cell, _ := excelize.CoordinatesToCellName(ci+1, totalRow)
		_ = f.SetCellValue(sheet, cell, val)
	}
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("%s%d", lastCol, totalRow), totalStyle)

	if len(r.Data) > 0 {
		_ = f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			YSplit:      headerRow,
			TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
			ActivePane:  "bottomLeft",
		})
	}
	_ = f.AutoFilter(sheet, fmt.Sprintf("A%d:%s%d", headerRow, lastCol, headerRow+len(r.Data)), []excelize.AutoFilterOptions{})

	widths := make([]float64, cols)
	for i, h := range r.Headers {
		widths[i] = float64(len([]rune(h))) + 3
	}
	for _, row := range r.Rows {
		for i, v := range row {
			if i < len(widths) {
				if l := float64(len([]rune(v))) + 2; l > widths[i] {
					widths[i] = l
				}
			}
		}
	}
	for i, w := range widths {
		if w > 58 {
			w = 58
		}
		name, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, name, name, w)
	}

	if err := addDetailSheet(f, r); err != nil {
		return err
	}
	if err := addSummarySheet(f, r, st); err != nil {
		return err
	}
	if idx, err := f.GetSheetIndex(sheet); err == nil {
		f.SetActiveSheet(idx)
	}
	return f.SaveAs(path)
}

func addDetailSheet(f *excelize.File, r report.Report) error {
	const name = "Detalle"
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	headers := []string{"Fecha", "Inicio", "Fin", "Horas", "Proyecto", "Directorio", "Sesión", "Session ID",
		"Input", "Output", "Reasoning", "Cache read", "Cache write", "Tokens", "Coste", "Editado"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(name, cell, h)
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"2D3E50"}, Pattern: 1},
	})
	last, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetCellStyle(name, "A1", last+"1", headerStyle)

	hourStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 2})
	tokenStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 3})
	customMoney := "0.0000"
	moneyStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 164, CustomNumFmt: &customMoney})
	var totalTokens int64
	var totalCost float64
	for i, s := range r.Raw {
		row := i + 2
		vals := []any{s.Day, s.Start.Format("15:04"), s.End.Format("15:04"), hoursOf(s),
			s.ProjectName, s.Project, s.Title, s.SessionID,
			s.Tokens.Input, s.Tokens.Output, s.Tokens.Reasoning, s.Tokens.CacheRead, s.Tokens.CacheWrite,
			s.Tokens.Total(), round4(s.Cost), boolText(s.Edited)}
		for ci, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(ci+1, row)
			_ = f.SetCellValue(name, cell, v)
			switch ci {
			case 3:
				_ = f.SetCellStyle(name, cell, cell, hourStyle)
			case 8, 9, 10, 11, 12, 13:
				_ = f.SetCellStyle(name, cell, cell, tokenStyle)
			case 14:
				_ = f.SetCellStyle(name, cell, cell, moneyStyle)
			}
		}
		totalTokens += s.Tokens.Total()
		totalCost += s.Cost
	}
	totalRow := len(r.Raw) + 2
	vals := []any{"TOTAL", "", "", hoursAny(r.Raw), "", "", "", "", "", "", "", "", "", totalTokens, round4(totalCost), ""}
	for ci, v := range vals {
		cell, _ := excelize.CoordinatesToCellName(ci+1, totalRow)
		_ = f.SetCellValue(name, cell, v)
	}
	totalStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E8EDF2"}, Pattern: 1}})
	_ = f.SetCellStyle(name, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("%s%d", last, totalRow), totalStyle)
	_ = f.SetColWidth(name, "A", "A", 12)
	_ = f.SetColWidth(name, "B", "C", 8)
	_ = f.SetColWidth(name, "D", "D", 8)
	_ = f.SetColWidth(name, "E", "E", 22)
	_ = f.SetColWidth(name, "F", "F", 40)
	_ = f.SetColWidth(name, "G", "G", 50)
	_ = f.SetColWidth(name, "H", "H", 34)
	_ = f.SetColWidth(name, "I", "O", 14)
	_ = f.SetColWidth(name, "P", "P", 9)
	_ = f.SetPanes(name, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	return nil
}

func addSummarySheet(f *excelize.File, r report.Report, st store.Stats) error {
	const name = "Resumen"
	if _, err := f.NewSheet(name); err != nil {
		return err
	}
	rows := [][]any{
		{"Sessions Control"},
		{"Generado", time.Now().Format("2006-01-02 15:04:05")},
		{"Vista", string(r.View)},
		{"Filtro", r.Filter},
		{"Fuente", st.Source},
		{"Sincronizado", st.SyncedAt.Format("2006-01-02 15:04:05")},
		{},
		{"Métrica", "Valor"},
		{"Sesiones", countSessionsExport(r.Raw)},
		{"Tramos (día/sesión)", len(r.Raw)},
		{"Horas trabajadas", hoursAny(r.Raw)},
		{"Horas únicas", uniqueHours(r.Raw)},
		{"Tokens totales", totalTokens(r.Raw)},
		{"Coste", round4(totalCost(r.Raw))},
		{},
		{"Resumen", r.Summary},
	}
	for i, row := range rows {
		for ci, v := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, i+1)
			_ = f.SetCellValue(name, cell, v)
		}
	}
	head, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"2D3E50"}, Pattern: 1},
	})
	_ = f.SetCellStyle(name, "A8", "B8", head)
	title, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 16}})
	_ = f.SetCellStyle(name, "A1", "B1", title)
	_ = f.SetColWidth(name, "A", "B", 30)
	return nil
}

func thinBorder() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "D0D7DE", Style: 1},
		{Type: "right", Color: "D0D7DE", Style: 1},
		{Type: "top", Color: "D0D7DE", Style: 1},
		{Type: "bottom", Color: "D0D7DE", Style: 1},
	}
}

func sheetName(v report.View) string {
	switch v {
	case report.ViewProjects:
		return "Proyectos"
	case report.ViewMonths:
		return "Meses"
	case report.ViewDays:
		return "Dias"
	default:
		return "Sesiones"
	}
}

func isMoneyColumn(h string) bool {
	return strings.EqualFold(h, "Coste") || strings.EqualFold(h, "Cost")
}

func hoursOf(s domain.Segment) float64 {
	return float64(s.End.Sub(s.Start)) / float64(time.Hour)
}

func hoursAny(segs []domain.Segment) float64 {
	var d time.Duration
	for _, s := range segs {
		d += s.End.Sub(s.Start)
	}
	return float64(d) / float64(time.Hour)
}

func uniqueHours(segs []domain.Segment) float64 {
	return float64(domain.UniqueDuration(segs)) / float64(time.Hour)
}

func totalTokens(segs []domain.Segment) int64 {
	var n int64
	for _, s := range segs {
		n += s.Tokens.Total()
	}
	return n
}

func totalCost(segs []domain.Segment) float64 {
	var f float64
	for _, s := range segs {
		f += s.Cost
	}
	return f
}

func countSessionsExport(segs []domain.Segment) int {
	seen := map[string]struct{}{}
	for _, s := range segs {
		seen[s.SessionID] = struct{}{}
	}
	return len(seen)
}

func boolText(b bool) string {
	if b {
		return "sí"
	}
	return ""
}

func round4(f float64) float64 { return float64(int64(f*10000+0.5)) / 10000 }
