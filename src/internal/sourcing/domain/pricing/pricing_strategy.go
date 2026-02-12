package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// PricingStrategyType: 料金計算戦略の種別
type PricingStrategyType string

const (
	PricingFlat       PricingStrategyType = "FLAT"
	PricingExpression PricingStrategyType = "EXPRESSION"
	PricingComposite  PricingStrategyType = "COMPOSITE"
)

type PricingStrategy interface {
	Calculate(ctx ShipmentContext) (shared.Money, error)
	Type() string
}
