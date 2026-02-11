package pricing

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// AppliedChargeItem: 適用された費目の計算結果
type AppliedChargeItem struct {
	LineItemID       uuid.UUID
	ChargeCode       string
	Category         string
	Amount           shared.Money
	ScopeDescription string // human-readable なスコープ説明
}

// SkippedChargeItem: スキップされた費目
type SkippedChargeItem struct {
	LineItemID uuid.UUID
	ChargeCode string
	Category   string
	Reason     string
}

// CurrencyTotal: 通貨別合計
type CurrencyTotal struct {
	Currency string
	Amount   shared.Money
}

// TariffCalculationResult: 料金表コスト計算の結果
type TariffCalculationResult struct {
	TariffID     uuid.UUID
	TariffName   string
	AppliedItems []AppliedChargeItem
	SkippedItems []SkippedChargeItem
	TotalAmounts []CurrencyTotal
}

// HasAppliedItems: 適用された費目があるか
func (r *TariffCalculationResult) HasAppliedItems() bool {
	return len(r.AppliedItems) > 0
}

// AppliedCount: 適用された費目の数
func (r *TariffCalculationResult) AppliedCount() int {
	return len(r.AppliedItems)
}

// SkippedCount: スキップされた費目の数
func (r *TariffCalculationResult) SkippedCount() int {
	return len(r.SkippedItems)
}
