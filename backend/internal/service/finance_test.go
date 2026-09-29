package service

import (
	"math"
	"testing"

	"github.com/go-kasir/backend/internal/repository"
)

func TestBuildFinanceMetricsSeparatesCapitalFromProfit(t *testing.T) {
	metrics := buildFinanceMetrics(40000, 16000, repository.FinanceTotals{
		CapitalInjections: 50000,
		AdditionalCapital: 10000,
		ProductPurchases:  20000,
		OperatingExpenses: 5000,
		InvestmentReturns: 3000,
		OtherIncome:       1000,
		OtherExpenses:     2000,
	})

	if metrics.InvestedCapital != 60000 {
		t.Fatalf("invested capital = %v, want 60000", metrics.InvestedCapital)
	}
	if metrics.TotalCashOutflow != 27000 {
		t.Fatalf("cash outflow = %v, want 27000", metrics.TotalCashOutflow)
	}
	if metrics.NetCashFlow != 77000 {
		t.Fatalf("net cash flow = %v, want 77000", metrics.NetCashFlow)
	}
	if metrics.NetProfit != 13000 {
		t.Fatalf("net profit = %v, want 13000", metrics.NetProfit)
	}
	if math.Abs(metrics.ROIPercent-21.6666667) > 0.001 {
		t.Fatalf("ROI = %v, want about 21.67", metrics.ROIPercent)
	}
}

func TestAssessLiquidity(t *testing.T) {
	tests := []struct {
		name       string
		cash       float64
		monthlyOut float64
		want       string
	}{
		{"negative cash", -1, 100, "Sangat kurang"},
		{"less than a quarter month", 24, 100, "Sangat kurang"},
		{"less than half month", 49, 100, "Kurang"},
		{"less than one month", 99, 100, "Cukup"},
		{"one to three months", 100, 100, "Baik"},
		{"three months or more", 300, 100, "Sangat baik"},
		{"no recorded outflow", 0, 0, "Cukup"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := assessLiquidity(test.cash, test.monthlyOut)
			if got.Status != test.want {
				t.Fatalf("status = %q, want %q", got.Status, test.want)
			}
		})
	}
}
