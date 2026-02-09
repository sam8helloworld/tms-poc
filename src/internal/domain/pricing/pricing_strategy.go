package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

type PricingStrategy interface {
	Calculate(ctx ShipmentContext) (shared.Money, error)
	Type() string
}
