package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/domain/calcparam"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/shopspring/decimal"
)

type PricingStrategy interface {
	Calculate(ctx calcparam.ShipmentContext) (shared.Money, error)
	Type() string
}

// Stage 1: Standard (Flat)
type FlatStrategy struct {
	Amount shared.Money
}

func (s *FlatStrategy) Type() string { return "FLAT" }
func (s *FlatStrategy) Calculate(ctx calcparam.ShipmentContext) (shared.Money, error) {
	// 単価 * 数量
	return s.Amount.Multiply(ctx.Quantity), nil
}

// Stage 2: Dynamic (CEL Expression)
// "max(weight, volume*167) * rate" などの複雑な式に対応
type CelExpressionStrategy struct {
	Formula  string
	Currency string
}

func (s *CelExpressionStrategy) Type() string { return "CEL" }
func (s *CelExpressionStrategy) Calculate(ctx calcparam.ShipmentContext) (shared.Money, error) {
	// ... Google CELによる評価ロジック ...
	return shared.Money{Amount: decimal.NewFromFloat(100.0), Currency: s.Currency}, nil
}

// Stage 3: Composite (Pipeline)
// 基本運賃 + BAF + CAF などを合成
type CompositeStrategy struct {
	Steps []PricingStrategy
}

func (s *CompositeStrategy) Type() string { return "COMPOSITE" }
func (s *CompositeStrategy) Calculate(ctx calcparam.ShipmentContext) (shared.Money, error) {
	if len(s.Steps) == 0 {
		return shared.ZeroMoney("USD"), nil
	}

	first, err := s.Steps[0].Calculate(ctx)
	if err != nil {
		return shared.Money{}, err
	}
	total := first

	for _, step := range s.Steps[1:] {
		res, err := step.Calculate(ctx)
		if err != nil {
			return shared.Money{}, err
		}
		total, err = total.Add(res)
		if err != nil {
			return shared.Money{}, err
		}
	}
	return total, nil
}
