package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/nestorpons/sessions-control/internal/report"
	"github.com/nestorpons/sessions-control/internal/store"
)

func PDF(path string, r report.Report, st store.Stats) error {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(8, 8, 8)
	pdf.SetAutoPageBreak(false, 10)
	pdf.SetTitle("Sessions Control - "+r.Title, true)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("cp1252")
	pdf.SetFont("Helvetica", "B", 15)
	pdf.SetTextColor(35, 47, 62)
	pdf.CellFormat(0, 8, tr("Sessions Control · "+r.Title), "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 100, 110)
	pdf.CellFormat(0, 4.5, tr("Generado: "+time.Now().Format("2006-01-02 15:04:05")+"   ·   Filtro: "+r.Filter), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 4.5, tr("Fuente: "+st.Source), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 4.5, tr(r.Summary), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	weights := columnWeights(r.View)
	total := 0.0
	for _, w := range weights {
		total += w
	}
	const usable = 281.0
	widths := make([]float64, len(weights))
	for i, w := range weights {
		widths[i] = usable * w / total
	}

	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetFillColor(45, 62, 80)
		pdf.SetTextColor(255, 255, 255)
		for i, h := range r.Headers {
			pdf.CellFormat(widths[i], 7, tr(h), "", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}

	drawHeader()
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(30, 35, 40)
	for ri, row := range r.Rows {
		if pdf.GetY() > 188 {
			addFooter(pdf, r)
			pdf.AddPage()
			drawHeader()
			pdf.SetFont("Helvetica", "", 7.5)
			pdf.SetTextColor(30, 35, 40)
		}
		fill := ri%2 == 1
		if fill {
			pdf.SetFillColor(243, 246, 248)
		}
		for i, cell := range row {
			align := "L"
			if i < len(r.Headers) && isNumericHeader(r.Headers[i]) {
				align = "R"
			}
			pdf.CellFormat(widths[i], 6, tr(truncate(pdf, cell, widths[i]-2)), "", 0, align, fill, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(232, 237, 242)
	pdf.SetTextColor(20, 25, 30)
	for i, cell := range r.Totals {
		align := "L"
		if i < len(r.Headers) && isNumericHeader(r.Headers[i]) {
			align = "R"
		}
		pdf.CellFormat(widths[i], 7, tr(truncate(pdf, cell, widths[i]-2)), "", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
	addFooter(pdf, r)

	return pdf.OutputFileAndClose(path)
}

func addFooter(pdf *fpdf.Fpdf, r report.Report) {
	pdf.SetY(-12)
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(120, 130, 140)
	pdf.CellFormat(0, 5, fmt.Sprintf("Página %d · %s", pdf.PageNo(), r.Summary), "", 0, "R", false, 0, "")
	pdf.SetY(-1)
}

func truncate(pdf *fpdf.Fpdf, s string, max float64) string {
	if pdf.GetStringWidth(s) <= max {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if pdf.GetStringWidth(string(runes)+"…") <= max {
			return string(runes) + "…"
		}
	}
	return ""
}

func isNumericHeader(h string) bool {
	switch strings.ToLower(h) {
	case "horas", "únicO", "unico", "tokens", "coste", "sesiones", "días", "dias":
		return true
	}
	return false
}

func columnWeights(v report.View) []float64 {
	switch v {
	case report.ViewProjects:
		return []float64{4.5, 1.5, 1.5, 1.6, 1.2, 1.6, 1.6, 2.3, 1.7}
	case report.ViewMonths:
		return []float64{3.0, 2.0, 1.6, 2.0, 2.0, 2.6, 2.0}
	case report.ViewDays:
		return []float64{3.0, 2.0, 2.0, 2.0, 2.6, 2.0}
	default:
		return []float64{2.1, 1.3, 1.3, 1.3, 3.4, 7.4, 2.2, 1.6}
	}
}
