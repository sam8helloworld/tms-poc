package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// CompositeStrategy: Stage 3 - Composite (Pipeline)
// 基本運賃 + BAF + CAF などを合成
type CompositeStrategy struct {
	Steps []PricingStrategy
}

func (s *CompositeStrategy) Type() string { return "COMPOSITE" }

func (s *CompositeStrategy) Calculate(ctx ShipmentContext) (shared.Money, error) {
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
