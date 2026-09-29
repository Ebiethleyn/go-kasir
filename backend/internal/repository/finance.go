package repository

import (
	"errors"
	"time"

	"github.com/go-kasir/backend/internal/model"
	"gorm.io/gorm"
)

type FinanceRepo struct{ db *gorm.DB }

func NewFinanceRepo(db *gorm.DB) *FinanceRepo { return &FinanceRepo{db: db} }

type FinanceTotals struct {
	CapitalInjections float64 `json:"capital_injections"`
	AdditionalCapital float64 `json:"additional_capital"`
	ProductPurchases  float64 `json:"product_purchases"`
	OperatingExpenses float64 `json:"operating_expenses"`
	InvestmentReturns float64 `json:"investment_returns"`
	OtherIncome       float64 `json:"other_income"`
	OtherExpenses     float64 `json:"other_expenses"`
}

func (r *FinanceRepo) Totals(from, to *time.Time) (FinanceTotals, error) {
	var totals FinanceTotals
	query := r.db.Model(&model.FinanceEntry{}).Select(`
		COALESCE(SUM(CASE WHEN entry_type = 'capital_injection' THEN amount ELSE 0 END), 0) AS capital_injections,
		COALESCE(SUM(CASE WHEN entry_type = 'additional_capital' THEN amount ELSE 0 END), 0) AS additional_capital,
		COALESCE(SUM(CASE WHEN entry_type = 'product_purchase' THEN amount ELSE 0 END), 0) AS product_purchases,
		COALESCE(SUM(CASE WHEN entry_type = 'operating_expense' THEN amount ELSE 0 END), 0) AS operating_expenses,
		COALESCE(SUM(CASE WHEN entry_type = 'investment_return' THEN amount ELSE 0 END), 0) AS investment_returns,
		COALESCE(SUM(CASE WHEN entry_type = 'other_income' THEN amount ELSE 0 END), 0) AS other_income,
		COALESCE(SUM(CASE WHEN entry_type = 'other_expense' THEN amount ELSE 0 END), 0) AS other_expenses`)
	if from != nil {
		query = query.Where("occurred_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("occurred_at <= ?", *to)
	}
	err := query.Scan(&totals).Error
	return totals, err
}

func (r *FinanceRepo) List(from, to *time.Time, limit int) ([]model.FinanceEntry, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	query := r.db.Model(&model.FinanceEntry{})
	if from != nil {
		query = query.Where("occurred_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("occurred_at <= ?", *to)
	}
	var entries []model.FinanceEntry
	err := query.Order("occurred_at DESC, id DESC").Limit(limit).Find(&entries).Error
	return entries, err
}

func (r *FinanceRepo) Create(entry *model.FinanceEntry) error {
	return r.db.Create(entry).Error
}

func (r *FinanceRepo) Delete(id uint, ownerID uint) error {
	res := r.db.Where("id = ? AND created_by = ?", id, ownerID).Delete(&model.FinanceEntry{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("catatan tidak ditemukan atau akses ditolak")
	}
	return nil
}
