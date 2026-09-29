package repository

import (
	"time"

	"github.com/go-kasir/backend/internal/model"
	"gorm.io/gorm"
)

type OwnerRepo struct{ db *gorm.DB }

func NewOwnerRepo(db *gorm.DB) *OwnerRepo { return &OwnerRepo{db: db} }

func (r *OwnerRepo) FindByUsername(username string) (*model.Owner, error) {
	var o model.Owner
	err := r.db.Where("username = ?", username).First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OwnerRepo) FindByID(id uint) (*model.Owner, error) {
	var o model.Owner
	err := r.db.First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OwnerRepo) Update(o *model.Owner) error {
	return r.db.Save(o).Error
}

func (r *OwnerRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&model.Owner{}).Count(&n).Error
	return n, err
}

func (r *OwnerRepo) Create(o *model.Owner) error {
	return r.db.Create(o).Error
}

// ---- Category ----

type CategoryRepo struct{ db *gorm.DB }

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) List() ([]model.Category, error) {
	var list []model.Category
	err := r.db.Order("name asc").Find(&list).Error
	return list, err
}

func (r *CategoryRepo) FindByID(id uint) (*model.Category, error) {
	var c model.Category
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepo) Create(c *model.Category) error { return r.db.Create(c).Error }
func (r *CategoryRepo) Update(c *model.Category) error { return r.db.Save(c).Error }
func (r *CategoryRepo) Delete(id uint) error {
	return r.db.Delete(&model.Category{}, id).Error
}

// ---- Product ----

type ProductRepo struct{ db *gorm.DB }

func NewProductRepo(db *gorm.DB) *ProductRepo { return &ProductRepo{db: db} }

type ProductFilter struct {
	Search     string
	CategoryID *uint
	LowStock   bool
	Page       int
	Limit      int
}

func (r *ProductRepo) List(f ProductFilter) ([]model.Product, int64, error) {
	q := r.db.Model(&model.Product{}).Preload("Category")
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name LIKE ? OR sku LIKE ? OR barcode LIKE ?", like, like, like)
	}
	if f.CategoryID != nil {
		q = q.Where("category_id = ?", *f.CategoryID)
	}
	if f.LowStock {
		q = q.Where("stock <= min_stock")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	var list []model.Product
	err := q.Order("name asc").Offset((f.Page - 1) * f.Limit).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

func (r *ProductRepo) FindByID(id uint) (*model.Product, error) {
	var p model.Product
	err := r.db.Preload("Category").First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) FindBySKU(sku string) (*model.Product, error) {
	var p model.Product
	err := r.db.Where("sku = ?", sku).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) FindByBarcodeOrSKU(code string) (*model.Product, error) {
	var p model.Product
	err := r.db.Where("barcode = ? OR sku = ?", code, code).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) Search(q string, limit int) ([]model.Product, error) {
	var list []model.Product
	like := "%" + q + "%"
	err := r.db.Where("is_active = 1 AND (name LIKE ? OR sku LIKE ? OR barcode LIKE ?)", like, like, like).
		Limit(limit).Order("name asc").Find(&list).Error
	return list, err
}

func (r *ProductRepo) Create(p *model.Product) error { return r.db.Create(p).Error }
func (r *ProductRepo) Update(p *model.Product) error { return r.db.Save(p).Error }
func (r *ProductRepo) Delete(id uint) error {
	return r.db.Delete(&model.Product{}, id).Error
}

func (r *ProductRepo) LowStock(limit int) ([]model.Product, error) {
	var list []model.Product
	err := r.db.Where("stock <= min_stock AND is_active = 1").Order("stock asc").Limit(limit).Find(&list).Error
	return list, err
}

func (r *ProductRepo) All() ([]model.Product, error) {
	var list []model.Product
	err := r.db.Preload("Category").Order("name asc").Find(&list).Error
	return list, err
}

// ---- Stock ----

type StockRepo struct{ db *gorm.DB }

func NewStockRepo(db *gorm.DB) *StockRepo { return &StockRepo{db: db} }

func (r *StockRepo) Create(m *model.StockMovement) error { return r.db.Create(m).Error }

func (r *StockRepo) List(productID *uint, from, to *time.Time, page, limit int) ([]model.StockMovement, int64, error) {
	q := r.db.Model(&model.StockMovement{}).Preload("Product")
	if productID != nil {
		q = q.Where("product_id = ?", *productID)
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}
	var total int64
	_ = q.Count(&total)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	var list []model.StockMovement
	err := q.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&list).Error
	return list, total, err
}

// ---- Customer ----

type CustomerRepo struct{ db *gorm.DB }

func NewCustomerRepo(db *gorm.DB) *CustomerRepo { return &CustomerRepo{db: db} }

type CustomerListItem struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Phone            string    `json:"phone"`
	Email            string    `json:"email"`
	Address          string    `json:"address"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	TransactionCount int64     `json:"transaction_count"`
	TransactionTotal float64   `json:"transaction_total"`
}

func (r *CustomerRepo) List(search, sortBy string, page, limit int) ([]CustomerListItem, int64, error) {
	q := r.db.Model(&model.Customer{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name LIKE ? OR phone LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	order := "customers.name ASC"
	switch sortBy {
	case "transactions_desc":
		order = "transaction_count DESC, customers.name ASC"
	case "transactions_asc":
		order = "transaction_count ASC, customers.name ASC"
	}
	var list []CustomerListItem
	err := q.Select(`customers.id, customers.name, customers.phone, customers.email,
		customers.address, customers.notes, customers.created_at, customers.updated_at,
		(SELECT COUNT(*) FROM transactions
		 WHERE transactions.customer_id = customers.id AND transactions.status = 'completed') AS transaction_count,
		(SELECT COALESCE(SUM(transactions.total), 0) FROM transactions
		 WHERE transactions.customer_id = customers.id AND transactions.status = 'completed') AS transaction_total`).
		Order(order).Offset((page - 1) * limit).Limit(limit).Scan(&list).Error
	return list, total, err
}

func (r *CustomerRepo) FindByID(id uint) (*model.Customer, error) {
	var c model.Customer
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepo) Create(c *model.Customer) error { return r.db.Create(c).Error }
func (r *CustomerRepo) Update(c *model.Customer) error { return r.db.Save(c).Error }
func (r *CustomerRepo) Delete(id uint) error {
	return r.db.Delete(&model.Customer{}, id).Error
}

func (r *CustomerRepo) All() ([]model.Customer, error) {
	var list []model.Customer
	err := r.db.Order("name asc").Find(&list).Error
	return list, err
}

// ---- Transaction ----

type TransactionRepo struct{ db *gorm.DB }

func NewTransactionRepo(db *gorm.DB) *TransactionRepo { return &TransactionRepo{db: db} }

type TrxFilter struct {
	From          *time.Time
	To            *time.Time
	Status        string
	PaymentMethod string
	Search        string
	CustomerID    *uint
	Page          int
	Limit         int
}

func (r *TransactionRepo) List(f TrxFilter) ([]model.Transaction, int64, error) {
	q := r.db.Model(&model.Transaction{}).Preload("Customer").Preload("Items")
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.PaymentMethod != "" {
		q = q.Where("payment_method = ?", f.PaymentMethod)
	}
	if f.CustomerID != nil {
		q = q.Where("customer_id = ?", *f.CustomerID)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("invoice_no LIKE ?", like)
	}
	var total int64
	_ = q.Count(&total)
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	var list []model.Transaction
	err := q.Order("created_at desc").Offset((f.Page - 1) * f.Limit).Limit(f.Limit).Find(&list).Error
	return list, total, err
}

func (r *TransactionRepo) FindByID(id uint) (*model.Transaction, error) {
	var t model.Transaction
	err := r.db.Preload("Customer").Preload("Items").Preload("Items.Product").First(&t, id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TransactionRepo) CreateWithItems(tx *gorm.DB, t *model.Transaction) error {
	return tx.Create(t).Error
}

func (r *TransactionRepo) Update(t *model.Transaction) error {
	return r.db.Save(t).Error
}

func (r *TransactionRepo) CountToday(from, to time.Time) (int64, float64, error) {
	var result struct {
		Cnt   int64
		Total float64
	}
	err := r.db.Model(&model.Transaction{}).
		Select("COUNT(*) as cnt, COALESCE(SUM(total),0) as total").
		Where("status = ? AND created_at >= ? AND created_at <= ?", "completed", from, to).
		Scan(&result).Error
	return result.Cnt, result.Total, err
}

func (r *TransactionRepo) DB() *gorm.DB { return r.db }

// ---- Settings ----

type SettingRepo struct{ db *gorm.DB }

func NewSettingRepo(db *gorm.DB) *SettingRepo { return &SettingRepo{db: db} }

func (r *SettingRepo) Get() (*model.Setting, error) {
	var s model.Setting
	err := r.db.First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SettingRepo) Save(s *model.Setting) error {
	return r.db.Save(s).Error
}

func (r *SettingRepo) Create(s *model.Setting) error {
	return r.db.Create(s).Error
}

// ---- Activity ----

type ActivityRepo struct{ db *gorm.DB }

func NewActivityRepo(db *gorm.DB) *ActivityRepo { return &ActivityRepo{db: db} }

func (r *ActivityRepo) Create(a *model.ActivityLog) error {
	return r.db.Create(a).Error
}

func (r *ActivityRepo) List(page, limit int) ([]model.ActivityLog, int64, error) {
	var total int64
	r.db.Model(&model.ActivityLog{}).Count(&total)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	var list []model.ActivityLog
	err := r.db.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&list).Error
	return list, total, err
}
