package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-kasir/backend/internal/model"
	"github.com/go-kasir/backend/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionService struct {
	trx      *repository.TransactionRepo
	products *repository.ProductRepo
	settings *repository.SettingRepo
	logs     *repository.ActivityRepo
	db       *gorm.DB
}

func NewTransactionService(
	trx *repository.TransactionRepo,
	products *repository.ProductRepo,
	settings *repository.SettingRepo,
	logs *repository.ActivityRepo,
	db *gorm.DB,
) *TransactionService {
	return &TransactionService{trx: trx, products: products, settings: settings, logs: logs, db: db}
}

type CartItemInput struct {
	ProductID      uint    `json:"product_id"`
	Quantity       int     `json:"quantity"`
	DiscountAmount float64 `json:"discount_amount"`
}

type CreateTransactionInput struct {
	CustomerID      *uint           `json:"customer_id"`
	Items           []CartItemInput `json:"items"`
	DiscountAmount  float64         `json:"discount_amount"`
	DiscountPercent float64         `json:"discount_percent"`
	TaxPercent      *float64        `json:"tax_percent"`
	PaidAmount      float64         `json:"paid_amount"`
	PaymentMethod   string          `json:"payment_method"`
	Notes           string          `json:"notes"`
}

func (s *TransactionService) Create(ownerID uint, in CreateTransactionInput) (*model.Transaction, error) {
	if len(in.Items) == 0 {
		return nil, errors.New("keranjang kosong")
	}
	method := in.PaymentMethod
	if method != "cash" && method != "card" && method != "qris" {
		return nil, errors.New("metode pembayaran harus cash, card, atau qris")
	}

	settings, err := s.settings.Get()
	if err != nil {
		return nil, errors.New("pengaturan toko belum tersedia")
	}
	taxPct := settings.TaxPercent
	if in.TaxPercent != nil {
		taxPct = *in.TaxPercent
	}

	var result *model.Transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var subtotal float64
		var items []model.TransactionItem

		for _, ci := range in.Items {
			if ci.Quantity <= 0 {
				return errors.New("kuantitas harus > 0")
			}
			var p model.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, ci.ProductID).Error; err != nil {
				return fmt.Errorf("produk #%d tidak ditemukan", ci.ProductID)
			}
			if !p.IsActive {
				return fmt.Errorf("produk %s tidak aktif", p.Name)
			}
			if p.Stock < ci.Quantity {
				return fmt.Errorf("stok %s tidak cukup (tersedia %d)", p.Name, p.Stock)
			}
			line := p.SellPrice*float64(ci.Quantity) - ci.DiscountAmount
			if line < 0 {
				line = 0
			}
			items = append(items, model.TransactionItem{
				ProductID:      p.ID,
				ProductName:    p.Name,
				ProductSKU:     p.SKU,
				Quantity:       ci.Quantity,
				BuyPrice:       p.BuyPrice,
				SellPrice:      p.SellPrice,
				DiscountAmount: ci.DiscountAmount,
				Subtotal:       line,
			})
			subtotal += line

			before := p.Stock
			after := before - ci.Quantity
			if err := tx.Model(&p).Update("stock", after).Error; err != nil {
				return err
			}
			mov := model.StockMovement{
				ProductID: p.ID, Type: "sale", Quantity: ci.Quantity,
				StockBefore: before, StockAfter: after,
				ReferenceType: "transaction", CreatedBy: ownerID,
				Notes: "Penjualan",
			}
			if err := tx.Create(&mov).Error; err != nil {
				return err
			}
		}

		disc := in.DiscountAmount
		if in.DiscountPercent > 0 {
			disc += subtotal * in.DiscountPercent / 100
		}
		afterDisc := subtotal - disc
		if afterDisc < 0 {
			afterDisc = 0
		}
		taxAmt := afterDisc * taxPct / 100
		total := afterDisc + taxAmt

		paid := in.PaidAmount
		if method != "cash" {
			paid = total
		}
		if paid < total {
			return errors.New("jumlah bayar kurang dari total")
		}
		change := paid - total

		invoice := fmt.Sprintf("%s-%s-%04d", settings.InvoicePrefix, time.Now().Format("20060102"), time.Now().Unix()%10000)

		trx := &model.Transaction{
			InvoiceNo:       invoice,
			CustomerID:      in.CustomerID,
			OwnerID:         ownerID,
			Subtotal:        subtotal,
			DiscountAmount:  disc,
			DiscountPercent: in.DiscountPercent,
			TaxAmount:       taxAmt,
			TaxPercent:      taxPct,
			Total:           total,
			PaidAmount:      paid,
			ChangeAmount:    change,
			PaymentMethod:   method,
			Status:          "completed",
			Notes:           in.Notes,
			Items:           items,
		}
		if err := tx.Create(trx).Error; err != nil {
			return err
		}
		// update stock movement reference
		_ = tx.Model(&model.StockMovement{}).
			Where("reference_type = ? AND reference_id IS NULL AND created_by = ? AND type = ?", "transaction", ownerID, "sale").
			Where("created_at >= ?", time.Now().Add(-1*time.Minute)).
			Update("reference_id", trx.ID)

		result = trx
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.trx.FindByID(result.ID)
}

func (s *TransactionService) List(f repository.TrxFilter) ([]model.Transaction, int64, error) {
	return s.trx.List(f)
}

func (s *TransactionService) Get(id uint) (*model.Transaction, error) {
	return s.trx.FindByID(id)
}

func (s *TransactionService) Void(id uint, ownerID uint, reason, ip string) error {
	if reason == "" {
		return errors.New("alasan void wajib diisi")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var t model.Transaction
		if err := tx.Preload("Items").First(&t, id).Error; err != nil {
			return errors.New("transaksi tidak ditemukan")
		}
		if t.Status == "voided" {
			return errors.New("transaksi sudah dibatalkan")
		}
		now := time.Now()
		t.Status = "voided"
		t.VoidReason = reason
		t.VoidedAt = &now
		if err := tx.Save(&t).Error; err != nil {
			return err
		}
		for _, item := range t.Items {
			var p model.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, item.ProductID).Error; err != nil {
				continue
			}
			before := p.Stock
			after := before + item.Quantity
			_ = tx.Model(&p).Update("stock", after).Error
			_ = tx.Create(&model.StockMovement{
				ProductID: p.ID, Type: "void", Quantity: item.Quantity,
				StockBefore: before, StockAfter: after,
				ReferenceType: "transaction", ReferenceID: &t.ID,
				Notes: "Void: " + reason, CreatedBy: ownerID,
			}).Error
		}
		eid := t.ID
		_ = s.logs.Create(&model.ActivityLog{
			OwnerID: ownerID, Action: "void_transaction", EntityType: "transaction",
			EntityID: &eid, Details: reason, IPAddress: ip,
		})
		return nil
	})
}

// ---- Customer ----

type CustomerService struct {
	repo *repository.CustomerRepo
}

func NewCustomerService(r *repository.CustomerRepo) *CustomerService {
	return &CustomerService{repo: r}
}

func (s *CustomerService) List(search, sortBy string, page, limit int) ([]repository.CustomerListItem, int64, error) {
	return s.repo.List(search, sortBy, page, limit)
}

func (s *CustomerService) Get(id uint) (*model.Customer, error) {
	return s.repo.FindByID(id)
}

func (s *CustomerService) Create(c *model.Customer) error {
	if c.Name == "" {
		return errors.New("nama pelanggan wajib")
	}
	return s.repo.Create(c)
}

func (s *CustomerService) Update(id uint, upd *model.Customer) (*model.Customer, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("pelanggan tidak ditemukan")
	}
	c.Name = upd.Name
	c.Phone = upd.Phone
	c.Email = upd.Email
	c.Address = upd.Address
	c.Notes = upd.Notes
	if err := s.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CustomerService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// ---- Settings ----

type SettingService struct {
	repo *repository.SettingRepo
}

func NewSettingService(r *repository.SettingRepo) *SettingService {
	return &SettingService{repo: r}
}

func (s *SettingService) Get() (*model.Setting, error) {
	return s.repo.Get()
}

func (s *SettingService) Update(upd *model.Setting) (*model.Setting, error) {
	cur, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	cur.StoreName = upd.StoreName
	cur.StoreAddress = upd.StoreAddress
	cur.StorePhone = upd.StorePhone
	cur.InvoicePrefix = upd.InvoicePrefix
	cur.TaxPercent = upd.TaxPercent
	cur.DefaultMinStock = upd.DefaultMinStock
	cur.ReceiptWidthMM = upd.ReceiptWidthMM
	cur.CurrencySymbol = upd.CurrencySymbol
	if upd.StoreLogo != "" {
		cur.StoreLogo = upd.StoreLogo
	}
	if err := s.repo.Save(cur); err != nil {
		return nil, err
	}
	return cur, nil
}

// ---- Dashboard ----

type DashboardService struct {
	trx     *repository.TransactionRepo
	report  *repository.ReportRepo
	product *repository.ProductRepo
}

func NewDashboardService(trx *repository.TransactionRepo, report *repository.ReportRepo, product *repository.ProductRepo) *DashboardService {
	return &DashboardService{trx: trx, report: report, product: product}
}

func (s *DashboardService) Summary() (map[string]interface{}, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24*time.Hour - time.Nanosecond)

	cnt, total, err := s.trx.CountToday(start, end)
	if err != nil {
		return nil, err
	}
	totalProfit, err := s.report.TotalProductProfit()
	if err != nil {
		return nil, err
	}
	top, _ := s.report.TopProducts(start, end, 5, false)
	low, _ := s.product.LowStock(10)

	weekStart := start.AddDate(0, 0, -6)
	trend, _ := s.report.SalesTrend(weekStart, end, "day")
	byCat, _ := s.report.SalesByCategory(weekStart, end)
	periods, err := s.periodMetrics(now)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"today_sales":        total,
		"total_profit":       totalProfit,
		"today_transactions": cnt,
		"top_products":       top,
		"low_stock":          low,
		"sales_trend_7d":     trend,
		"category_sales_7d":  byCat,
		"period_metrics":     periods,
	}, nil
}

type dashboardPeriod struct {
	Period             string                 `json:"period"`
	Label              string                 `json:"label"`
	TransactionCount   int64                  `json:"transaction_count"`
	Profit             float64                `json:"profit"`
	TopQuantityProduct *repository.TopProduct `json:"top_quantity_product"`
	TopTrxProduct      *repository.TopProduct `json:"top_transaction_product"`
	TopRevenueProduct  *repository.TopProduct `json:"top_revenue_product"`
	TopPriceProduct    *repository.TopProduct `json:"top_price_product"`
}

func (s *DashboardService) periodMetrics(now time.Time) ([]dashboardPeriod, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := dayStart.AddDate(0, 0, -((int(dayStart.Weekday()) + 6) % 7))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	quarterMonth := time.Month(((int(now.Month())-1)/3)*3 + 1)
	quarterStart := time.Date(now.Year(), quarterMonth, 1, 0, 0, 0, 0, now.Location())
	halfYearMonth := time.January
	if now.Month() > time.June {
		halfYearMonth = time.July
	}
	periods := []struct {
		period string
		label  string
		start  time.Time
	}{
		{"day", "Hari ini", dayStart},
		{"week", "Minggu ini", weekStart},
		{"month", "Bulan ini", monthStart},
		{"quarter", "Kuartal ini", quarterStart},
		{"semester", "Semester ini", time.Date(now.Year(), halfYearMonth, 1, 0, 0, 0, 0, now.Location())},
		{"year", "Tahun ini", time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())},
	}

	result := make([]dashboardPeriod, 0, len(periods))
	for _, period := range periods {
		summary, err := s.report.SalesSummary(period.start, now)
		if err != nil {
			return nil, err
		}
		profit, err := s.report.ProductProfit(period.start, now)
		if err != nil {
			return nil, err
		}
		quantityProducts, err := s.report.TopProducts(period.start, now, 1, false)
		if err != nil {
			return nil, err
		}
		revenueProducts, err := s.report.TopProductsByRevenue(period.start, now, 1)
		if err != nil {
			return nil, err
		}
		priceProducts, err := s.report.TopProductsByPrice(period.start, now, 1)
		if err != nil {
			return nil, err
		}
		transactionProducts, err := s.report.TopProductsByTransactions(period.start, now, 1)
		if err != nil {
			return nil, err
		}
		result = append(result, dashboardPeriod{
			Period:             period.period,
			Label:              period.label,
			TransactionCount:   summary.TransactionCount,
			Profit:             profit,
			TopQuantityProduct: firstProduct(quantityProducts),
			TopTrxProduct:      firstProduct(transactionProducts),
			TopRevenueProduct:  firstProduct(revenueProducts),
			TopPriceProduct:    firstProduct(priceProducts),
		})
	}
	return result, nil
}

func firstProduct(products []repository.TopProduct) *repository.TopProduct {
	if len(products) == 0 {
		return nil
	}
	return &products[0]
}

// ---- Report ----

type ReportService struct {
	report   *repository.ReportRepo
	trx      *repository.TransactionRepo
	product  *repository.ProductRepo
	stock    *repository.StockRepo
	customer *repository.CustomerRepo
}

func NewReportService(
	report *repository.ReportRepo,
	trx *repository.TransactionRepo,
	product *repository.ProductRepo,
	stock *repository.StockRepo,
	customer *repository.CustomerRepo,
) *ReportService {
	return &ReportService{report: report, trx: trx, product: product, stock: stock, customer: customer}
}

func ParseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	loc := time.Local
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	to := from.Add(24*time.Hour - time.Nanosecond)
	var err error
	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, loc)
		if err != nil {
			return from, to, errors.New("format tanggal from harus YYYY-MM-DD")
		}
	}
	if toStr != "" {
		t, err := time.ParseInLocation("2006-01-02", toStr, loc)
		if err != nil {
			return from, to, errors.New("format tanggal to harus YYYY-MM-DD")
		}
		to = t.Add(24*time.Hour - time.Nanosecond)
	}
	return from, to, nil
}

func (s *ReportService) Sales(from, to time.Time, groupBy string) (map[string]interface{}, error) {
	sum, err := s.report.SalesSummary(from, to)
	if err != nil {
		return nil, err
	}
	trend, _ := s.report.SalesTrend(from, to, groupBy)
	byCat, _ := s.report.SalesByCategory(from, to)
	byPay, _ := s.report.SalesByPayment(from, to)
	return map[string]interface{}{
		"summary": sum, "trend": trend, "by_category": byCat, "by_payment": byPay,
		"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
	}, nil
}

func (s *ReportService) Products(from, to time.Time) (map[string]interface{}, error) {
	top, _ := s.report.TopProducts(from, to, 20, false)
	slow, _ := s.report.TopProducts(from, to, 20, true)
	byCat, _ := s.report.SalesByCategory(from, to)
	stock, _ := s.product.All()
	return map[string]interface{}{
		"top_products": top, "slow_products": slow, "by_category": byCat, "current_stock": stock,
		"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
	}, nil
}

func (s *ReportService) ProfitLoss(from, to time.Time) (map[string]interface{}, error) {
	sum, err := s.report.SalesSummary(from, to)
	if err != nil {
		return nil, err
	}
	trend, _ := s.report.SalesTrend(from, to, "day")
	return map[string]interface{}{
		"summary": sum, "trend": trend,
		"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"),
	}, nil
}

func (s *ReportService) Transactions(f repository.TrxFilter) ([]model.Transaction, int64, error) {
	return s.trx.List(f)
}

func (s *ReportService) CustomerHistory(customerID uint, from, to time.Time) (map[string]interface{}, error) {
	c, err := s.customer.FindByID(customerID)
	if err != nil {
		return nil, errors.New("pelanggan tidak ditemukan")
	}
	purchases, err := s.report.CustomerPurchases(customerID, from, to)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"customer": c, "purchases": purchases}, nil
}
