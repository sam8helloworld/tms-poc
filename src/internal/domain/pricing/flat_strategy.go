package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// FlatStrategy: Stage 1 - Standard (Flat)
type FlatStrategy struct {
	Amount shared.Money
}

func (s *FlatStrategy) Type() string { return "FLAT" }

func (s *FlatStrategy) Calculate(ctx ShipmentContext) (shared.Money, error) {
	// 単価 * 数量
	return s.Amount.Multiply(ctx.Quantity), nil
}
