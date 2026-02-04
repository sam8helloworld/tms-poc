package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/domain/context"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/shopspring/decimal"
)

type PricingStrategy interface {
	Calculate(ctx context.ShipmentContext) (shared.Money, error)
	Type() string
}

// Stage 1: Standard (Flat)
type FlatStrategy struct {
	Amount shared.Money
}

func (s *FlatStrategy) Type() string { return "FLAT" }
func (s *FlatStrategy) Calculate(ctx context.ShipmentContext) (shared.Money, error) {
	// 単価 * 数量
	val := s.Amount.Amount.Mul(ctx.Quantity)
	return shared.Money{Amount: val, Currency: s.Amount.Currency}, nil
}

// Stage 2: Dynamic (CEL Expression)
// "max(weight, volume*167) * rate" などの複雑な式に対応
type CelExpressionStrategy struct {
	Formula  string
	Currency string
}

func (s *CelExpressionStrategy) Type() string { return "CEL" }
func (s *CelExpressionStrategy) Calculate(ctx context.ShipmentContext) (shared.Money, error) {
	// ... Google CELによる評価ロジック ...
	return shared.Money{Amount: decimal.NewFromFloat(100.0), Currency: s.Currency}, nil
}

// Stage 3: Composite (Pipeline)
// 基本運賃 + BAF + CAF などを合成
type CompositeStrategy struct {
	Steps []PricingStrategy
}

func (s *CompositeStrategy) Type() string { return "COMPOSITE" }
func (s *CompositeStrategy) Calculate(ctx context.ShipmentContext) (shared.Money, error) {
	total := decimal.Zero
	var currency string
	for _, step := range s.Steps {
		res, _ := step.Calculate(ctx)
		total = total.Add(res.Amount)
		currency = res.Currency
	}
	return shared.Money{Amount: total, Currency: currency}, nil
}
