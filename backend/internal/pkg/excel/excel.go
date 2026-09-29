package excel

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-kasir/backend/internal/model"
	"github.com/xuri/excelize/v2"
)

func newFile(sheet string, headers []string, rows [][]interface{}) ([]byte, error) {
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", sheet)
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for r, row := range rows {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, val)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SalesReport(from, to time.Time, summary map[string]interface{}, trend []map[string]interface{}) ([]byte, error) {
	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Penjualan")
	_ = f.SetCellValue("Penjualan", "A1", "Laporan Penjualan")
	_ = f.SetCellValue("Penjualan", "A2", fmt.Sprintf("%s s/d %s", from.Format("2006-01-02"), to.Format("2006-01-02")))
	row := 4
	for k, v := range summary {
		_ = f.SetCellValue("Penjualan", fmt.Sprintf("A%d", row), k)
		_ = f.SetCellValue("Penjualan", fmt.Sprintf("B%d", row), v)
		row++
	}
	row += 2
	_ = f.SetCellValue("Penjualan", fmt.Sprintf("A%d", row), "Periode")
	_ = f.SetCellValue("Penjualan", fmt.Sprintf("B%d", row), "Total")
	_ = f.SetCellValue("Penjualan", fmt.Sprintf("C%d", row), "Jumlah Trx")
	row++
	for _, t := range trend {
		_ = f.SetCellValue("Penjualan", fmt.Sprintf("A%d", row), t["period"])
		_ = f.SetCellValue("Penjualan", fmt.Sprintf("B%d", row), t["total"])
		_ = f.SetCellValue("Penjualan", fmt.Sprintf("C%d", row), t["count"])
		row++
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ProductsExport(products []model.Product) ([]byte, error) {
	rows := make([][]interface{}, 0, len(products))
	for _, p := range products {
		cat := ""
		if p.Category != nil {
			cat = p.Category.Name
		}
		rows = append(rows, []interface{}{p.SKU, p.Barcode, p.Name, cat, p.BuyPrice, p.SellPrice, p.Stock, p.MinStock, p.Unit})
	}
	return newFile("Produk", []string{"SKU", "Barcode", "Nama", "Kategori", "Harga Beli", "Harga Jual", "Stok", "Min Stok", "Satuan"}, rows)
}

func CategoriesExport(cats []model.Category) ([]byte, error) {
	rows := make([][]interface{}, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, []interface{}{c.ID, c.Name, c.Description})
	}
	return newFile("Kategori", []string{"ID", "Nama", "Deskripsi"}, rows)
}

func CustomersExport(list []model.Customer) ([]byte, error) {
	rows := make([][]interface{}, 0, len(list))
	for _, c := range list {
		rows = append(rows, []interface{}{c.ID, c.Name, c.Phone, c.Email, c.Address})
	}
	return newFile("Pelanggan", []string{"ID", "Nama", "Telepon", "Email", "Alamat"}, rows)
}

func TransactionsExport(list []model.Transaction) ([]byte, error) {
	rows := make([][]interface{}, 0, len(list))
	for _, t := range list {
		rows = append(rows, []interface{}{
			t.InvoiceNo, t.CreatedAt.Format("2006-01-02 15:04"), t.Total, t.PaymentMethod, t.Status, t.DiscountAmount, t.TaxAmount,
		})
	}
	return newFile("Transaksi", []string{"Invoice", "Tanggal", "Total", "Metode", "Status", "Diskon", "Pajak"}, rows)
}

func GenericTable(sheet string, headers []string, rows [][]interface{}) ([]byte, error) {
	return newFile(sheet, headers, rows)
}

func ParseProductImport(data []byte) ([]model.Product, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("file excel kosong")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	var out []model.Product
	for i, row := range rows {
		if i == 0 {
			continue // header
		}
		if len(row) < 3 {
			continue
		}
		p := model.Product{SKU: row[0], Name: row[2], Unit: "pcs", IsActive: true, MinStock: 5}
		if len(row) > 1 {
			p.Barcode = row[1]
		}
		if len(row) > 4 {
			fmt.Sscanf(row[4], "%f", &p.BuyPrice)
		}
		if len(row) > 5 {
			fmt.Sscanf(row[5], "%f", &p.SellPrice)
		}
		if len(row) > 6 {
			fmt.Sscanf(row[6], "%d", &p.Stock)
		}
		if len(row) > 8 && row[8] != "" {
			p.Unit = row[8]
		}
		out = append(out, p)
	}
	return out, nil
}
