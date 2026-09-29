package model

import (
	"time"

	"gorm.io/gorm"
)

// Owner — akun login tunggal (role Owner/Admin).
type Owner struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	FullName     string         `gorm:"size:100;not null" json:"full_name"`
	Email        string         `gorm:"size:100" json:"email"`
	Phone        string         `gorm:"size:30" json:"phone"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Category struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Description string         `gorm:"size:255" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Product struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CategoryID  *uint          `gorm:"index" json:"category_id"`
	Category    *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	SKU         string         `gorm:"size:50;uniqueIndex;not null" json:"sku"`
	Barcode     string         `gorm:"size:50;index" json:"barcode"`
	Name        string         `gorm:"size:150;not null;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	BuyPrice    float64        `gorm:"type:decimal(15,2);not null;default:0" json:"buy_price"`
	SellPrice   float64        `gorm:"type:decimal(15,2);not null;default:0" json:"sell_price"`
	Stock       int            `gorm:"not null;default:0" json:"stock"`
	MinStock    int            `gorm:"not null;default:5" json:"min_stock"`
	Unit        string         `gorm:"size:20;default:pcs" json:"unit"`
	ImagePath   string         `gorm:"size:255" json:"image_path"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// StockMovement — histori keluar/masuk stok.
type StockMovement struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProductID     uint      `gorm:"index;not null" json:"product_id"`
	Product       *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Type          string    `gorm:"size:20;not null;index" json:"type"` // in, out, adjust, sale, void
	Quantity      int       `gorm:"not null" json:"quantity"`
	StockBefore   int       `gorm:"not null" json:"stock_before"`
	StockAfter    int       `gorm:"not null" json:"stock_after"`
	ReferenceType string    `gorm:"size:30" json:"reference_type"` // transaction, manual, import
	ReferenceID   *uint     `json:"reference_id"`
	Notes         string    `gorm:"size:255" json:"notes"`
	CreatedBy     uint      `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type Customer struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null;index" json:"name"`
	Phone     string         `gorm:"size:30" json:"phone"`
	Email     string         `gorm:"size:100" json:"email"`
	Address   string         `gorm:"size:255" json:"address"`
	Notes     string         `gorm:"size:255" json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Transaction struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	InvoiceNo       string            `gorm:"size:50;uniqueIndex;not null" json:"invoice_no"`
	CustomerID      *uint             `gorm:"index" json:"customer_id"`
	Customer        *Customer         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	OwnerID         uint              `gorm:"not null" json:"owner_id"`
	Subtotal        float64           `gorm:"type:decimal(15,2);not null;default:0" json:"subtotal"`
	DiscountAmount  float64           `gorm:"type:decimal(15,2);not null;default:0" json:"discount_amount"`
	DiscountPercent float64           `gorm:"type:decimal(5,2);not null;default:0" json:"discount_percent"`
	TaxAmount       float64           `gorm:"type:decimal(15,2);not null;default:0" json:"tax_amount"`
	TaxPercent      float64           `gorm:"type:decimal(5,2);not null;default:0" json:"tax_percent"`
	Total           float64           `gorm:"type:decimal(15,2);not null;default:0" json:"total"`
	PaidAmount      float64           `gorm:"type:decimal(15,2);not null;default:0" json:"paid_amount"`
	ChangeAmount    float64           `gorm:"type:decimal(15,2);not null;default:0" json:"change_amount"`
	PaymentMethod   string            `gorm:"size:20;not null;index" json:"payment_method"`           // cash, card, qris
	Status          string            `gorm:"size:20;not null;default:completed;index" json:"status"` // completed, voided
	VoidReason      string            `gorm:"size:255" json:"void_reason"`
	VoidedAt        *time.Time        `json:"voided_at"`
	Notes           string            `gorm:"size:255" json:"notes"`
	Items           []TransactionItem `gorm:"foreignKey:TransactionID" json:"items,omitempty"`
	CreatedAt       time.Time         `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type TransactionItem struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TransactionID  uint      `gorm:"index;not null" json:"transaction_id"`
	ProductID      uint      `gorm:"index;not null" json:"product_id"`
	Product        *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductName    string    `gorm:"size:150;not null" json:"product_name"`
	ProductSKU     string    `gorm:"size:50" json:"product_sku"`
	Quantity       int       `gorm:"not null" json:"quantity"`
	BuyPrice       float64   `gorm:"type:decimal(15,2);not null;default:0" json:"buy_price"`
	SellPrice      float64   `gorm:"type:decimal(15,2);not null;default:0" json:"sell_price"`
	DiscountAmount float64   `gorm:"type:decimal(15,2);not null;default:0" json:"discount_amount"`
	Subtotal       float64   `gorm:"type:decimal(15,2);not null;default:0" json:"subtotal"`
	CreatedAt      time.Time `json:"created_at"`
}

type Setting struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	StoreName       string    `gorm:"size:150;not null;default:Toko Saya" json:"store_name"`
	StoreAddress    string    `gorm:"size:255" json:"store_address"`
	StorePhone      string    `gorm:"size:30" json:"store_phone"`
	StoreLogo       string    `gorm:"size:255" json:"store_logo"`
	InvoicePrefix   string    `gorm:"size:20;default:INV" json:"invoice_prefix"`
	TaxPercent      float64   `gorm:"type:decimal(5,2);default:0" json:"tax_percent"`
	DefaultMinStock int       `gorm:"default:5" json:"default_min_stock"`
	ReceiptWidthMM  int       `gorm:"default:80" json:"receipt_width_mm"` // 58 or 80
	CurrencySymbol  string    `gorm:"size:10;default:Rp" json:"currency_symbol"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type FinanceEntry struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	EntryType   string    `gorm:"size:30;not null;index" json:"entry_type"`
	Amount      float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	OccurredAt  time.Time `gorm:"not null;index" json:"occurred_at"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedBy   uint      `gorm:"not null;index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type ActivityLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	OwnerID    uint      `gorm:"index" json:"owner_id"`
	Action     string    `gorm:"size:50;not null;index" json:"action"`
	EntityType string    `gorm:"size:50" json:"entity_type"`
	EntityID   *uint     `json:"entity_id"`
	Details    string    `gorm:"type:text" json:"details"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}
