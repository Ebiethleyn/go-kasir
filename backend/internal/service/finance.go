package service

import (
	"errors"
	"strings"
	"time"

	"github.com/go-kasir/backend/internal/model"
	"github.com/go-kasir/backend/internal/repository"
)

var financeEntryTypes = map[string]bool{
	"capital_injection":  true,
	"additional_capital": true,
	"product_purchase":   true,
	"operating_expense":  true,
	"investment_return":  true,
	"other_income":       true,
	"other_expense":      true,
}

type FinanceService struct {
	entries *repository.FinanceRepo
	reports *repository.ReportRepo
}

func NewFinanceService(entries *repository.FinanceRepo, reports *repository.ReportRepo) *FinanceService {
	return &FinanceService{entries: entries, reports: reports}
}

type FinanceEntryInput struct {
	EntryType   string  `json:"entry_type"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
}

type FinanceMetrics struct {
	SalesRevenue      float64 `json:"sales_revenue"`
	GrossProfit       float64 `json:"gross_profit"`
	CapitalInjections float64 `json:"capital_injections"`
	AdditionalCapital float64 `json:"additional_capital"`
	InvestedCapital   float64 `json:"invested_capital"`
	ProductPurchases  float64 `json:"product_purchases"`
	OperatingExpenses float64 `json:"operating_expenses"`
	InvestmentReturns float64 `json:"investment_returns"`
	OtherIncome       float64 `json:"other_income"`
	OtherExpenses     float64 `json:"other_expenses"`
	TotalCashOutflow  float64 `json:"total_cash_outflow"`
	NetCashFlow       float64 `json:"net_cash_flow"`
	NetProfit         float64 `json:"net_profit"`
	ROIPercent        float64 `json:"roi_percent"`
}

type FinancePeriod struct {
	Period string `json:"period"`
	Label  string `json:"label"`
	FinanceMetrics
}

type LiquidityAssessment struct {
	Status                string  `json:"status"`
	CashBalance           float64 `json:"cash_balance"`
	AverageMonthlyOutflow float64 `json:"average_monthly_outflow"`
	CoverageMonths        float64 `json:"coverage_months"`
	Basis                 string  `json:"basis"`
}

func (s *FinanceService) CreateEntry(ownerID uint, input FinanceEntryInput) (*model.FinanceEntry, error) {
	input.EntryType = strings.TrimSpace(input.EntryType)
	input.Description = strings.TrimSpace(input.Description)
	if !financeEntryTypes[input.EntryType] {
		return nil, errors.New("jenis arus kas tidak valid")
	}
	if input.Amount <= 0 {
		return nil, errors.New("nominal harus lebih besar dari 0")
	}
	if len(input.Description) > 255 {
		return nil, errors.New("keterangan maksimal 255 karakter")
	}
	occurredAt := time.Now()
	if input.Date != "" {
		parsed, err := time.ParseInLocation("2006-01-02", input.Date, time.Local)
		if err != nil {
			return nil, errors.New("format tanggal harus YYYY-MM-DD")
		}
		now := time.Now()
		occurredAt = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.Local)
	}
	entry := &model.FinanceEntry{
		EntryType: input.EntryType, Amount: input.Amount, OccurredAt: occurredAt,
		Description: input.Description, CreatedBy: ownerID,
	}
	if err := s.entries.Create(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *FinanceService) DeleteEntry(id uint, ownerID uint) error {
	return s.entries.Delete(id, ownerID)
}

func (s *FinanceService) ListEntries(from, to time.Time, limit int) ([]model.FinanceEntry, error) {
	return s.entries.List(&from, &to, limit)
}

func (s *FinanceService) Summary(from, to time.Time) (map[string]interface{}, error) {
	selected, err := s.metrics(&from, &to)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	epoch := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.Local)
	allTime, err := s.metrics(&epoch, &now)
	if err != nil {
		return nil, err
	}
	selected.ROIPercent = calculateROI(selected.NetProfit, allTime.InvestedCapital)

	ninetyDaysAgo := now.AddDate(0, 0, -90)
	recentEntries, err := s.entries.Totals(&ninetyDaysAgo, &now)
	if err != nil {
		return nil, err
	}
	averageMonthlyOutflow := (recentEntries.ProductPurchases + recentEntries.OperatingExpenses + recentEntries.OtherExpenses) / 3
	liquidity := assessLiquidity(allTime.NetCashFlow, averageMonthlyOutflow)

	periods, err := s.periodMetrics(now, allTime.InvestedCapital)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"selected_range": selected,
		"all_time":       allTime,
		"periods":        periods,
		"liquidity":      liquidity,
	}, nil
}

func (s *FinanceService) metrics(from, to *time.Time) (FinanceMetrics, error) {
	totals, err := s.entries.Totals(from, to)
	if err != nil {
		return FinanceMetrics{}, err
	}
	var salesRevenue, grossProfit float64
	if from != nil && to != nil {
		summary, err := s.reports.SalesSummary(*from, *to)
		if err != nil {
			return FinanceMetrics{}, err
		}
		salesRevenue = summary.TotalRevenue
		grossProfit, err = s.reports.ProductProfit(*from, *to)
		if err != nil {
			return FinanceMetrics{}, err
		}
	}
	return buildFinanceMetrics(salesRevenue, grossProfit, totals), nil
}

func buildFinanceMetrics(salesRevenue, grossProfit float64, totals repository.FinanceTotals) FinanceMetrics {
	investedCapital := totals.CapitalInjections + totals.AdditionalCapital
	cashOutflow := totals.ProductPurchases + totals.OperatingExpenses + totals.OtherExpenses
	cashInflow := salesRevenue + investedCapital + totals.InvestmentReturns + totals.OtherIncome
	netProfit := grossProfit - totals.OperatingExpenses - totals.OtherExpenses + totals.InvestmentReturns + totals.OtherIncome
	return FinanceMetrics{
		SalesRevenue: salesRevenue, GrossProfit: grossProfit,
		CapitalInjections: totals.CapitalInjections, AdditionalCapital: totals.AdditionalCapital,
		InvestedCapital: investedCapital, ProductPurchases: totals.ProductPurchases,
		OperatingExpenses: totals.OperatingExpenses, InvestmentReturns: totals.InvestmentReturns,
		OtherIncome: totals.OtherIncome, OtherExpenses: totals.OtherExpenses,
		TotalCashOutflow: cashOutflow, NetCashFlow: cashInflow - cashOutflow,
		NetProfit: netProfit, ROIPercent: calculateROI(netProfit, investedCapital),
	}
}

func calculateROI(netProfit, investedCapital float64) float64 {
	if investedCapital <= 0 {
		return 0
	}
	return netProfit / investedCapital * 100
}

func assessLiquidity(cashBalance, averageMonthlyOutflow float64) LiquidityAssessment {
	assessment := LiquidityAssessment{
		CashBalance:           cashBalance,
		AverageMonthlyOutflow: averageMonthlyOutflow,
		Basis:                 "Daya tahan = saldo estimasi dibagi rata-rata pengeluaran bulanan 90 hari terakhir. Di bawah 0,25 bulan: sangat kurang; di bawah 0,5: kurang; di bawah 1: cukup; di bawah 3: baik; 3 bulan atau lebih: sangat baik.",
	}
	if cashBalance < 0 {
		assessment.Status = "Sangat kurang"
		return assessment
	}
	if averageMonthlyOutflow <= 0 {
		assessment.Status = "Cukup"
		assessment.Basis = "Belum ada pengeluaran tercatat yang cukup untuk menghitung daya tahan kas."
		return assessment
	}
	assessment.CoverageMonths = cashBalance / averageMonthlyOutflow
	switch {
	case assessment.CoverageMonths < 0.25:
		assessment.Status = "Sangat kurang"
	case assessment.CoverageMonths < 0.5:
		assessment.Status = "Kurang"
	case assessment.CoverageMonths < 1:
		assessment.Status = "Cukup"
	case assessment.CoverageMonths < 3:
		assessment.Status = "Baik"
	default:
		assessment.Status = "Sangat baik"
	}
	return assessment
}

func (s *FinanceService) periodMetrics(now time.Time, investedCapital float64) ([]FinancePeriod, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	week := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	quarter := time.Date(now.Year(), time.Month(((int(now.Month())-1)/3)*3+1), 1, 0, 0, 0, 0, now.Location())
	halfYearMonth := time.January
	if now.Month() > time.June {
		halfYearMonth = time.July
	}
	periods := []struct {
		key   string
		label string
		from  time.Time
	}{
		{"day", "Hari ini", today},
		{"week", "Minggu ini", week},
		{"month", "Bulan ini", month},
		{"quarter", "Kuartal ini", quarter},
		{"semester", "Semester ini", time.Date(now.Year(), halfYearMonth, 1, 0, 0, 0, 0, now.Location())},
		{"year", "Tahun ini", time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())},
	}
	result := make([]FinancePeriod, 0, len(periods))
	for _, period := range periods {
		to := now
		metrics, err := s.metrics(&period.from, &to)
		if err != nil {
			return nil, err
		}
		metrics.ROIPercent = calculateROI(metrics.NetProfit, investedCapital)
		result = append(result, FinancePeriod{Period: period.key, Label: period.label, FinanceMetrics: metrics})
	}
	return result, nil
}
