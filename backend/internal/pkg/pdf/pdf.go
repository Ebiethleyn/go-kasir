package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kasir/backend/internal/model"
	"github.com/jung-kurt/gofpdf"
)

func formatRp(v float64) string {
	amount := fmt.Sprintf("%.0f", v)
	sign := ""
	if strings.HasPrefix(amount, "-") {
		sign = "-"
		amount = strings.TrimPrefix(amount, "-")
	}
	for i := len(amount) - 3; i > 0; i -= 3 {
		amount = amount[:i] + "." + amount[i:]
	}
	return "Rp " + sign + amount
}

// Receipt generates a compact, thermal-style receipt PDF (58mm or 80mm).
func Receipt(trx *model.Transaction, store *model.Setting) ([]byte, error) {
	widthMM := float64(80)
	if store.ReceiptWidthMM == 58 {
		widthMM = 58
	}
	lineChars := int((widthMM - 6) / 1.7)
	pageHeight := 108.0 + float64(len(trx.Items))*9
	for _, item := range trx.Items {
		nameLines := (utf8.RuneCountInString(item.ProductName) + lineChars - 1) / lineChars
		if nameLines > 1 {
			pageHeight += float64(nameLines-1) * 4
		}
	}
	if trx.DiscountAmount > 0 {
		pageHeight += 4
	}
	if trx.TaxAmount > 0 {
		pageHeight += 4
	}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size:    gofpdf.SizeType{Wd: widthMM, Ht: pageHeight},
	})
	pdf.SetMargins(3, 3, 3)
	pdf.SetAutoPageBreak(false, 3)
	pdf.AddPage()
	innerWidth := widthMM - 6
	pdf.SetTextColor(25, 47, 39)
	pdf.SetFont("Arial", "B", 12)
	pdf.MultiCell(widthMM-6, 5, store.StoreName, "", "C", false)
	pdf.SetFont("Arial", "", 8)
	if store.StoreAddress != "" {
		pdf.SetTextColor(83, 97, 90)
		pdf.MultiCell(innerWidth, 4, store.StoreAddress, "", "C", false)
	}
	if store.StorePhone != "" {
		pdf.SetTextColor(83, 97, 90)
		pdf.CellFormat(innerWidth, 4, store.StorePhone, "", 1, "C", false, 0, "")
	}
	pdf.Ln(2)
	pdf.SetFillColor(25, 82, 65)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(innerWidth, 6, "BUKTI PEMBAYARAN", "", 1, "C", true, 0, "")
	pdf.Ln(2)
	pdf.SetTextColor(35, 48, 42)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(innerWidth*.62, 4, "INVOICE", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(innerWidth*.38, 4, trx.InvoiceNo, "", 1, "R", false, 0, "")
	pdf.CellFormat(innerWidth, 4, trx.CreatedAt.Format("02/01/2006 15:04"), "", 1, "L", false, 0, "")
	method := map[string]string{"cash": "Tunai", "card": "Kartu", "qris": "QRIS"}[trx.PaymentMethod]
	if method == "" {
		method = trx.PaymentMethod
	}
	pdf.CellFormat(innerWidth, 4, "Pembayaran: "+method, "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.SetDrawColor(178, 195, 186)
	pdf.SetLineWidth(.25)
	pdf.Line(3, pdf.GetY(), widthMM-3, pdf.GetY())
	pdf.Ln(2)

	for _, it := range trx.Items {
		pdf.SetTextColor(25, 47, 39)
		pdf.SetFont("Arial", "B", 8)
		pdf.MultiCell(innerWidth, 4, it.ProductName, "", "L", false)
		pdf.SetTextColor(83, 97, 90)
		pdf.SetFont("Arial", "", 7)
		line := fmt.Sprintf("%d x %s", it.Quantity, formatRp(it.SellPrice))
		pdf.CellFormat(innerWidth*.64, 4, line, "", 0, "L", false, 0, "")
		pdf.SetTextColor(35, 48, 42)
		pdf.CellFormat(innerWidth*.36, 4, formatRp(it.Subtotal), "", 1, "R", false, 0, "")
		pdf.Ln(1)
	}
	pdf.SetDrawColor(178, 195, 186)
	pdf.Line(3, pdf.GetY(), widthMM-3, pdf.GetY())
	pdf.Ln(2)
	row := func(label string, val float64) {
		pdf.CellFormat(innerWidth*.55, 4, label, "", 0, "L", false, 0, "")
		pdf.CellFormat(innerWidth*.45, 4, formatRp(val), "", 1, "R", false, 0, "")
	}
	pdf.SetTextColor(83, 97, 90)
	pdf.SetFont("Arial", "", 8)
	row("Subtotal", trx.Subtotal)
	if trx.DiscountAmount > 0 {
		row("Diskon", trx.DiscountAmount)
	}
	if trx.TaxAmount > 0 {
		row(fmt.Sprintf("Pajak (%.0f%%)", trx.TaxPercent), trx.TaxAmount)
	}
	pdf.Ln(1)
	pdf.SetDrawColor(25, 82, 65)
	pdf.SetLineWidth(.45)
	pdf.Line(3, pdf.GetY(), widthMM-3, pdf.GetY())
	pdf.Ln(2)
	pdf.SetTextColor(25, 82, 65)
	pdf.SetFont("Arial", "B", 10)
	row("TOTAL", trx.Total)
	pdf.SetTextColor(35, 48, 42)
	pdf.SetFont("Arial", "", 8)
	row("Bayar", trx.PaidAmount)
	row("Kembali", trx.ChangeAmount)
	pdf.Ln(4)
	pdf.SetTextColor(25, 82, 65)
	pdf.SetFont("Arial", "B", 9)
	pdf.MultiCell(innerWidth, 4, "TERIMA KASIH", "", "C", false)
	pdf.SetTextColor(83, 97, 90)
	pdf.SetFont("Arial", "", 7)
	pdf.MultiCell(innerWidth, 3.5, "Simpan struk ini sebagai bukti transaksi.", "", "C", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SalesReport(title string, from, to time.Time, summary map[string]interface{}, rows [][]string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, title, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Periode: %s s/d %s", from.Format("02/01/2006"), to.Format("02/01/2006")), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 6, "Dicetak: "+time.Now().Format("02/01/2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(4)

	if summary != nil {
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(0, 6, "Ringkasan", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		for k, v := range summary {
			pdf.CellFormat(0, 5, fmt.Sprintf("%s: %v", k, v), "", 1, "L", false, 0, "")
		}
		pdf.Ln(3)
	}

	if len(rows) > 0 {
		pdf.SetFont("Arial", "B", 9)
		colW := 186.0 / float64(len(rows[0]))
		for _, h := range rows[0] {
			pdf.CellFormat(colW, 6, h, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Arial", "", 8)
		for i, row := range rows {
			if i == 0 {
				continue
			}
			for _, cell := range row {
				pdf.CellFormat(colW, 5, cell, "1", 0, "L", false, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
