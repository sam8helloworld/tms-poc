package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// CelExpressionStrategy: Stage 2 - Dynamic (CEL Expression)
// "max(weight, volume*167) * rate" などの複雑な式に対応
type CelExpressionStrategy struct {
	Formula  string
	Currency string
}

func (s *CelExpressionStrategy) Type() string { return "CEL" }

func (s *CelExpressionStrategy) Calculate(ctx ShipmentContext) (shared.Money, error) {
	// ... Google CELによる評価ロジック ...
	return shared.Money{Amount: decimal.NewFromFloat(100.0), Currency: s.Currency}, nil
}
