package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-kasir/backend/internal/middleware"
	"github.com/go-kasir/backend/internal/model"
	"github.com/go-kasir/backend/internal/pkg/backup"
	"github.com/go-kasir/backend/internal/pkg/excel"
	"github.com/go-kasir/backend/internal/pkg/pdf"
	"github.com/go-kasir/backend/internal/pkg/response"
	"github.com/go-kasir/backend/internal/repository"
	"github.com/go-kasir/backend/internal/service"
)

// ---- Transactions ----

func (h *Handlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var in service.CreateTransactionInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	trx, err := h.Trx.Create(middleware.GetOwnerID(r), in)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, "transaksi berhasil", trx)
}

func (h *Handlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	f := repository.TrxFilter{
		From: &from, To: &to,
		Status:        r.URL.Query().Get("status"),
		PaymentMethod: r.URL.Query().Get("payment_method"),
		Search:        r.URL.Query().Get("search"),
		Page:          atoiDefault(r.URL.Query().Get("page"), 1),
		Limit:         atoiDefault(r.URL.Query().Get("limit"), 20),
	}
	list, total, err := h.Trx.List(f)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total})
}

func (h *Handlers) GetTransaction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	t, err := h.Trx.Get(uint(id))
	if err != nil {
		response.NotFound(w, "transaksi tidak ditemukan")
		return
	}
	response.OK(w, "ok", t)
}

func (h *Handlers) VoidTransaction(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	if err := h.Trx.Void(uint(id), middleware.GetOwnerID(r), req.Reason, clientIP(r)); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "transaksi dibatalkan", nil)
}

func (h *Handlers) ReceiptPDF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	t, err := h.Trx.Get(uint(id))
	if err != nil {
		response.NotFound(w, "transaksi tidak ditemukan")
		return
	}
	store, err := h.Setting.Get()
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	data, err := pdf.Receipt(t, store)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"struk_%s.pdf\"", t.InvoiceNo))
	w.Write(data)
}

// ---- Customers ----

func (h *Handlers) ListCustomers(w http.ResponseWriter, r *http.Request) {
	list, total, err := h.Customer.List(r.URL.Query().Get("search"), r.URL.Query().Get("sort"), atoiDefault(r.URL.Query().Get("page"), 1), atoiDefault(r.URL.Query().Get("limit"), 20))
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total})
}

func (h *Handlers) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	c, err := h.Customer.Get(uint(id))
	if err != nil {
		response.NotFound(w, "pelanggan tidak ditemukan")
		return
	}
	response.OK(w, "ok", c)
}

func (h *Handlers) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var c model.Customer
	if err := decodeJSON(r, &c); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	if err := h.Customer.Create(&c); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, "pelanggan dibuat", c)
}

func (h *Handlers) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var c model.Customer
	if err := decodeJSON(r, &c); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	out, err := h.Customer.Update(uint(id), &c)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "pelanggan diperbarui", out)
}

func (h *Handlers) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.Customer.Delete(uint(id)); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "pelanggan dihapus", nil)
}

// ---- Settings ----

func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.Setting.Get()
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", s)
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s model.Setting
	if err := decodeJSON(r, &s); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	out, err := h.Setting.Update(&s)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "pengaturan disimpan", out)
}

// ---- Reports ----

func (h *Handlers) ReportSales(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "day"
	}
	data, err := h.Report.Sales(from, to, groupBy)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

func (h *Handlers) ReportProducts(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Report.Products(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

func (h *Handlers) ReportProfitLoss(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Report.ProfitLoss(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

func (h *Handlers) ReportTransactions(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	f := repository.TrxFilter{
		From: &from, To: &to,
		Status:        r.URL.Query().Get("status"),
		PaymentMethod: r.URL.Query().Get("payment_method"),
		Page:          atoiDefault(r.URL.Query().Get("page"), 1),
		Limit:         atoiDefault(r.URL.Query().Get("limit"), 100),
	}
	list, total, err := h.Report.Transactions(f)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total})
}

func (h *Handlers) ReportCustomer(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Report.CustomerHistory(uint(id), from, to)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

func (h *Handlers) ExportSalesPDF(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Report.Sales(from, to, "day")
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	sum := map[string]interface{}{}
	if s, ok := data["summary"].(*repository.SalesSummary); ok {
		sum["Total Omzet"] = formatMoney(s.TotalRevenue)
		sum["Laba Kotor"] = formatMoney(s.GrossProfit)
		sum["Jumlah Transaksi"] = s.TransactionCount
		sum["Rata-rata Trx"] = formatMoney(s.AvgTransaction)
	}
	rows := [][]string{{"Periode", "Total", "Jumlah"}}
	if trend, ok := data["trend"].([]repository.SalesTrendPoint); ok {
		for _, t := range trend {
			rows = append(rows, []string{t.Period, formatMoney(t.Total), strconv.FormatInt(t.Count, 10)})
		}
	}
	bin, err := pdf.SalesReport("Laporan Penjualan", from, to, sum, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writePDF(w, "laporan_penjualan.pdf", bin)
}

func (h *Handlers) ExportSalesExcel(w http.ResponseWriter, r *http.Request) {
	from, to, err := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	data, err := h.Report.Sales(from, to, "day")
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	sum := map[string]interface{}{}
	if s, ok := data["summary"].(*repository.SalesSummary); ok {
		sum["total_revenue"] = s.TotalRevenue
		sum["gross_profit"] = s.GrossProfit
		sum["transaction_count"] = s.TransactionCount
		sum["avg_transaction"] = s.AvgTransaction
	}
	var trendMaps []map[string]interface{}
	if trend, ok := data["trend"].([]repository.SalesTrendPoint); ok {
		for _, t := range trend {
			trendMaps = append(trendMaps, map[string]interface{}{"period": t.Period, "total": t.Total, "count": t.Count})
		}
	}
	bin, err := excel.SalesReport(from, to, sum, trendMaps)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, "laporan_penjualan.xlsx", bin)
}

func (h *Handlers) ExportProductsPDF(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.Products(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	rows := [][]string{{"Produk", "SKU", "Qty", "Omzet"}}
	if top, ok := data["top_products"].([]repository.TopProduct); ok {
		for _, p := range top {
			rows = append(rows, []string{p.ProductName, p.ProductSKU, strconv.FormatInt(p.Quantity, 10), formatMoney(p.Revenue)})
		}
	}
	bin, err := pdf.SalesReport("Laporan Produk Terlaris", from, to, nil, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writePDF(w, "laporan_produk.pdf", bin)
}

func (h *Handlers) ExportProductsExcel(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.Products(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	var rows [][]interface{}
	if top, ok := data["top_products"].([]repository.TopProduct); ok {
		for _, p := range top {
			rows = append(rows, []interface{}{p.ProductName, p.ProductSKU, p.Quantity, p.Revenue})
		}
	}
	bin, err := excel.GenericTable("Produk", []string{"Nama", "SKU", "Qty", "Omzet"}, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, "laporan_produk.xlsx", bin)
}

func (h *Handlers) ExportProfitPDF(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.ProfitLoss(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	sum := map[string]interface{}{}
	if s, ok := data["summary"].(*repository.SalesSummary); ok {
		sum["Pendapatan"] = formatMoney(s.TotalRevenue)
		sum["HPP"] = formatMoney(s.TotalCOGS)
		sum["Laba Kotor"] = formatMoney(s.GrossProfit)
	}
	bin, err := pdf.SalesReport("Laporan Laba Rugi", from, to, sum, nil)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writePDF(w, "laporan_laba_rugi.pdf", bin)
}

func (h *Handlers) ExportProfitExcel(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.ProfitLoss(from, to)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	var rows [][]interface{}
	if s, ok := data["summary"].(*repository.SalesSummary); ok {
		rows = append(rows,
			[]interface{}{"Pendapatan", s.TotalRevenue},
			[]interface{}{"HPP", s.TotalCOGS},
			[]interface{}{"Laba Kotor", s.GrossProfit},
		)
	}
	bin, err := excel.GenericTable("LabaRugi", []string{"Pos", "Nilai"}, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, "laporan_laba_rugi.xlsx", bin)
}

func (h *Handlers) ExportTrxPDF(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	f := repository.TrxFilter{From: &from, To: &to, Page: 1, Limit: 500}
	list, _, err := h.Report.Transactions(f)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	rows := [][]string{{"Invoice", "Tanggal", "Total", "Metode", "Status"}}
	for _, t := range list {
		rows = append(rows, []string{t.InvoiceNo, t.CreatedAt.Format("02/01/2006 15:04"), formatMoney(t.Total), t.PaymentMethod, t.Status})
	}
	bin, err := pdf.SalesReport("Laporan Transaksi", from, to, nil, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writePDF(w, "laporan_transaksi.pdf", bin)
}

func (h *Handlers) ExportTrxExcel(w http.ResponseWriter, r *http.Request) {
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	f := repository.TrxFilter{From: &from, To: &to, Page: 1, Limit: 500}
	list, _, err := h.Report.Transactions(f)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	bin, err := excel.TransactionsExport(list)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, "laporan_transaksi.xlsx", bin)
}

func (h *Handlers) ExportCustomerPDF(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.CustomerHistory(uint(id), from, to)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	cust := data["customer"].(*model.Customer)
	rows := [][]string{{"Invoice", "Tanggal", "Total", "Metode", "Status"}}
	if purchases, ok := data["purchases"].([]map[string]interface{}); ok {
		for _, p := range purchases {
			rows = append(rows, []string{
				fmt.Sprint(p["invoice_no"]),
				fmt.Sprint(p["created_at"]),
				fmt.Sprint(p["total"]),
				fmt.Sprint(p["payment_method"]),
				fmt.Sprint(p["status"]),
			})
		}
	}
	bin, err := pdf.SalesReport("Histori Pelanggan: "+cust.Name, from, to, nil, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writePDF(w, "laporan_pelanggan.pdf", bin)
}

func (h *Handlers) ExportCustomerExcel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.Report.CustomerHistory(uint(id), from, to)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	var rows [][]interface{}
	if purchases, ok := data["purchases"].([]map[string]interface{}); ok {
		for _, p := range purchases {
			rows = append(rows, []interface{}{p["invoice_no"], p["created_at"], p["total"], p["payment_method"], p["status"]})
		}
	}
	bin, err := excel.GenericTable("Pelanggan", []string{"Invoice", "Tanggal", "Total", "Metode", "Status"}, rows)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, "laporan_pelanggan.xlsx", bin)
}

// ---- Backup & Master Export ----

func (h *Handlers) BackupNow(w http.ResponseWriter, r *http.Request) {
	data, fname, err := backup.Dump(h.Cfg)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	oid := middleware.GetOwnerID(r)
	_ = h.Logs.Create(&model.ActivityLog{
		OwnerID: oid, Action: "backup", EntityType: "database",
		Details: fname, IPAddress: clientIP(r),
	})
	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fname+"\"")
	w.Write(data)
}

func (h *Handlers) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		response.BadRequest(w, "upload gagal")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "field file wajib (.sql)")
		return
	}
	defer file.Close()
	_ = backup.SanitizeSQLFilename(header.Filename)
	if err := backup.Restore(h.Cfg, file); err != nil {
		response.Internal(w, err.Error())
		return
	}
	oid := middleware.GetOwnerID(r)
	_ = h.Logs.Create(&model.ActivityLog{
		OwnerID: oid, Action: "restore", EntityType: "database",
		Details: header.Filename, IPAddress: clientIP(r),
	})
	response.OK(w, "restore berhasil", nil)
}

func (h *Handlers) ExportMasterExcel(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	var bin []byte
	var err error
	var fname string
	switch typ {
	case "products":
		list, e := h.Products.All()
		err = e
		if err == nil {
			bin, err = excel.ProductsExport(list)
			fname = "master_produk.xlsx"
		}
	case "categories":
		list, e := h.Cats.List()
		err = e
		if err == nil {
			bin, err = excel.CategoriesExport(list)
			fname = "master_kategori.xlsx"
		}
	case "customers":
		list, e := h.Custs.All()
		err = e
		if err == nil {
			bin, err = excel.CustomersExport(list)
			fname = "master_pelanggan.xlsx"
		}
	case "transactions":
		from := time.Now().AddDate(-1, 0, 0)
		to := time.Now()
		list, _, e := h.Trxs.List(repository.TrxFilter{From: &from, To: &to, Page: 1, Limit: 10000})
		err = e
		if err == nil {
			bin, err = excel.TransactionsExport(list)
			fname = "master_transaksi.xlsx"
		}
	default:
		response.BadRequest(w, "type harus: products|categories|customers|transactions")
		return
	}
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	writeExcel(w, fname, bin)
}

func (h *Handlers) ActivityLogs(w http.ResponseWriter, r *http.Request) {
	list, total, err := h.Logs.List(atoiDefault(r.URL.Query().Get("page"), 1), atoiDefault(r.URL.Query().Get("limit"), 50))
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total})
}

func writePDF(w http.ResponseWriter, name string, data []byte) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Write(data)
}

func writeExcel(w http.ResponseWriter, name string, data []byte) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Write(data)
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.0f", v)
}
