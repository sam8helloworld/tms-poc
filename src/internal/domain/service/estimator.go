package service

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
	"github.com/sam8helloworld/tms-poc/internal/domain/context"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// FreightEstimator: 見積もり計算サービス (Domain Service)
type FreightEstimator struct{}

func (fe *FreightEstimator) Estimate(ctx context.ShipmentContext, tariff *commercial.Tariff) (EstimatedCost, error) {
	var lines []CalculatedLineItem

	// タリフ内の全LineItemを走査
	for _, item := range tariff.LineItems {
		// 1. 適用範囲(Scope)のチェック
		if item.Scope.IsApplicable(ctx) {
			// 2. 金額(Logic)の計算
			amt, err := item.Logic.Calculate(ctx)
			if err != nil {
				return EstimatedCost{}, err
			}

			lines = append(lines, CalculatedLineItem{
				ChargeCode: item.ChargeCode,
				Category:   item.Category,
				Amount:     amt,
			})
		}
	}

	return EstimatedCost{
		TariffID:  tariff.ID,
		LineItems: lines,
	}, nil
}

// EstimatedCost: 計算結果 (Result)
type EstimatedCost struct {
	TariffID  uuid.UUID
	LineItems []CalculatedLineItem
}

type CalculatedLineItem struct {
	ChargeCode string
	Category   string
	Amount     shared.Money
}
