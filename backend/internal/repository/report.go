package repository

import (
	"time"

	"gorm.io/gorm"
)

type ReportRepo struct{ db *gorm.DB }

func NewReportRepo(db *gorm.DB) *ReportRepo { return &ReportRepo{db: db} }

type SalesSummary struct {
	TotalRevenue     float64 `json:"total_revenue"`
	TotalCOGS        float64 `json:"total_cogs"`
	GrossProfit      float64 `json:"gross_profit"`
	TransactionCount int64   `json:"transaction_count"`
	AvgTransaction   float64 `json:"avg_transaction"`
	TotalItemsSold   int64   `json:"total_items_sold"`
}

type SalesTrendPoint struct {
	Period string  `json:"period"`
	Total  float64 `json:"total"`
	Count  int64   `json:"count"`
}

type CategorySales struct {
	CategoryID   *uint   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
	Quantity     int64   `json:"quantity"`
}

type PaymentBreakdown struct {
	PaymentMethod string  `json:"payment_method"`
	Total         float64 `json:"total"`
	Count         int64   `json:"count"`
}

type TopProduct struct {
	ProductID    uint    `json:"product_id"`
	ProductName  string  `json:"product_name"`
	ProductSKU   string  `json:"product_sku"`
	Quantity     int64   `json:"quantity"`
	Transactions int64   `json:"transactions"`
	Revenue      float64 `json:"revenue"`
	HighestPrice float64 `json:"highest_price"`
}

func (r *ReportRepo) SalesSummary(from, to time.Time) (*SalesSummary, error) {
	var s SalesSummary
	err := r.db.Raw(`
		SELECT
			COALESCE(SUM(t.total), 0) AS total_revenue,
			COALESCE(SUM(ti.cogs), 0) AS total_cogs,
			COALESCE(SUM(t.total), 0) - COALESCE(SUM(ti.cogs), 0) AS gross_profit,
			COUNT(DISTINCT t.id) AS transaction_count,
			CASE WHEN COUNT(DISTINCT t.id) = 0 THEN 0
				ELSE COALESCE(SUM(t.total), 0) / COUNT(DISTINCT t.id) END AS avg_transaction,
			COALESCE(SUM(ti.qty), 0) AS total_items_sold
		FROM transactions t
		LEFT JOIN (
			SELECT transaction_id,
				SUM(buy_price * quantity) AS cogs,
				SUM(quantity) AS qty
			FROM transaction_items
			GROUP BY transaction_id
		) ti ON ti.transaction_id = t.id
		WHERE t.status = 'completed' AND t.created_at >= ? AND t.created_at <= ?
	`, from, to).Scan(&s).Error
	return &s, err
}

func (r *ReportRepo) ProductProfit(from, to time.Time) (float64, error) {
	var profit float64
	err := r.db.Raw(`
		SELECT COALESCE(SUM((ti.sell_price - ti.buy_price) * ti.quantity), 0)
		FROM transaction_items ti
	JOIN transactions t ON t.id = ti.transaction_id
	WHERE t.status = 'completed' AND t.created_at >= ? AND t.created_at <= ?
	`, from, to).Scan(&profit).Error
	return profit, err
}

func (r *ReportRepo) TotalProductProfit() (float64, error) {
	var profit float64
	err := r.db.Raw(`
		SELECT COALESCE(SUM((ti.sell_price - ti.buy_price) * ti.quantity), 0)
		FROM transaction_items ti
		JOIN transactions t ON t.id = ti.transaction_id
		WHERE t.status = 'completed'
	`).Scan(&profit).Error
	return profit, err
}

func (r *ReportRepo) SalesTrend(from, to time.Time, groupBy string) ([]SalesTrendPoint, error) {
	var format string
	switch groupBy {
	case "week":
		format = "%x-W%v"
	case "month":
		format = "%Y-%m"
	case "year":
		format = "%Y"
	default:
		format = "%Y-%m-%d"
	}
	var points []SalesTrendPoint
	err := r.db.Raw(`
		SELECT DATE_FORMAT(created_at, ?) AS period,
			COALESCE(SUM(total), 0) AS total,
			COUNT(*) AS count
		FROM transactions
		WHERE status = 'completed' AND created_at >= ? AND created_at <= ?
		GROUP BY period
		ORDER BY period ASC
	`, format, from, to).Scan(&points).Error
	return points, err
}

func (r *ReportRepo) SalesByCategory(from, to time.Time) ([]CategorySales, error) {
	var list []CategorySales
	err := r.db.Raw(`
		SELECT p.category_id AS category_id,
			COALESCE(c.name, 'Tanpa Kategori') AS category_name,
			COALESCE(SUM(ti.subtotal), 0) AS total,
			COALESCE(SUM(ti.quantity), 0) AS quantity
		FROM transaction_items ti
		JOIN transactions t ON t.id = ti.transaction_id
		JOIN products p ON p.id = ti.product_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE t.status = 'completed' AND t.created_at >= ? AND t.created_at <= ?
		GROUP BY p.category_id, c.name
		ORDER BY total DESC
	`, from, to).Scan(&list).Error
	return list, err
}

func (r *ReportRepo) SalesByPayment(from, to time.Time) ([]PaymentBreakdown, error) {
	var list []PaymentBreakdown
	err := r.db.Raw(`
		SELECT payment_method,
			COALESCE(SUM(total), 0) AS total,
			COUNT(*) AS count
		FROM transactions
		WHERE status = 'completed' AND created_at >= ? AND created_at <= ?
		GROUP BY payment_method
		ORDER BY total DESC
	`, from, to).Scan(&list).Error
	return list, err
}

func (r *ReportRepo) TopProducts(from, to time.Time, limit int, orderAsc bool) ([]TopProduct, error) {
	order := "quantity DESC"
	if orderAsc {
		order = "quantity ASC"
	}
	return r.rankedProducts(from, to, limit, order)
}

func (r *ReportRepo) TopProductsByRevenue(from, to time.Time, limit int) ([]TopProduct, error) {
	return r.rankedProducts(from, to, limit, "revenue DESC")
}

func (r *ReportRepo) TopProductsByPrice(from, to time.Time, limit int) ([]TopProduct, error) {
	return r.rankedProducts(from, to, limit, "highest_price DESC")
}

func (r *ReportRepo) TopProductsByTransactions(from, to time.Time, limit int) ([]TopProduct, error) {
	return r.rankedProducts(from, to, limit, "transactions DESC")
}

func (r *ReportRepo) rankedProducts(from, to time.Time, limit int, order string) ([]TopProduct, error) {
	if limit < 1 {
		limit = 10
	}
	var list []TopProduct
	err := r.db.Raw(`
		SELECT ti.product_id AS product_id,
			ti.product_name AS product_name,
			ti.product_sku AS product_sku,
			COALESCE(SUM(ti.quantity), 0) AS quantity,
			COUNT(DISTINCT t.id) AS transactions,
			COALESCE(SUM(ti.subtotal), 0) AS revenue,
			COALESCE(MAX(ti.sell_price), 0) AS highest_price
		FROM transaction_items ti
		JOIN transactions t ON t.id = ti.transaction_id
		WHERE t.status = 'completed' AND t.created_at >= ? AND t.created_at <= ?
		GROUP BY ti.product_id, ti.product_name, ti.product_sku
		ORDER BY `+order+`, product_name ASC
		LIMIT ?
	`, from, to, limit).Scan(&list).Error
	return list, err
}

func (r *ReportRepo) CustomerPurchases(customerID uint, from, to time.Time) ([]map[string]interface{}, error) {
	type row struct {
		InvoiceNo     string    `json:"invoice_no"`
		Total         float64   `json:"total"`
		PaymentMethod string    `json:"payment_method"`
		Status        string    `json:"status"`
		CreatedAt     time.Time `json:"created_at"`
	}
	var rows []row
	err := r.db.Raw(`
		SELECT invoice_no, total, payment_method, status, created_at
		FROM transactions
		WHERE customer_id = ? AND created_at >= ? AND created_at <= ?
		ORDER BY created_at DESC
	`, customerID, from, to).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]interface{}{
			"invoice_no":     r.InvoiceNo,
			"total":          r.Total,
			"payment_method": r.PaymentMethod,
			"status":         r.Status,
			"created_at":     r.CreatedAt,
		})
	}
	return out, nil
}
