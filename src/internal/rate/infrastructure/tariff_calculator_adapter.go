package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/shopspring/decimal"
)

// TariffCalculatorAdapter: TariffCalculatorインターフェースの実装（ACLアダプタ）
// Sourcing BCのTariff.CalculateCharges()に委譲し、結果をRate BC固有の型に変換する。
// Rate BCのドメイン/アプリケーション層がSourcing BCに依存しないよう、ここで型変換を行う。
type TariffCalculatorAdapter struct {
	tariffRepo pricing.TariffRepository
}

// NewTariffCalculatorAdapter: TariffCalculatorAdapterのコンストラクタ
func NewTariffCalculatorAdapter(tariffRepo pricing.TariffRepository) *TariffCalculatorAdapter {
	return &TariffCalculatorAdapter{tariffRepo: tariffRepo}
}

// Calculate: TariffCalculatorインターフェースの実装
func (a *TariffCalculatorAdapter) Calculate(
	ctx context.Context,
	input domainrate.TariffCalculationInput,
) (*domainrate.TariffChargeResult, error) {
	// 1. Sourcing BCのTariffをロード
	tariff, err := a.tariffRepo.FindByID(ctx, input.TariffID)
	if err != nil {
		return nil, fmt.Errorf("tariff not found (ID: %s): %w", input.TariffID, err)
	}

	// 2. CalculationRequestを構築（Rate BCの型 → Sourcing BCの型への変換）
	calcReq, err := buildCalculationRequest(input)
	if err != nil {
		return nil, fmt.Errorf("failed to build calculation request: %w", err)
	}

	// 3. Tariff.CalculateCharges()を呼び出し
	calcResult, err := tariff.CalculateCharges(*calcReq)
	if err != nil {
		return nil, fmt.Errorf("tariff calculation failed: %w", err)
	}

	// 4. 結果から対象LineItemを抽出し、Rate BCの型に変換
	return extractResultForLineItem(calcResult, input.TariffLineItemID)
}

// buildCalculationRequest: Rate BCの入力からSourcing BCのCalculationRequestを構築（ACL変換）
func buildCalculationRequest(input domainrate.TariffCalculationInput) (*pricing.CalculationRequest, error) {
	// 課金重量の算出
	chargeableWeight := input.WeightKG
	volumetricWeight := input.VolumeM3.Mul(decimal.NewFromInt(1000))
	if volumetricWeight.GreaterThan(chargeableWeight) {
		chargeableWeight = volumetricWeight
	}

	summary := pricing.CargoSummary{
		TotalQuantity:         input.Quantity,
		TotalWeightKG:         input.WeightKG,
		TotalVolumeM3:         input.VolumeM3,
		ChargeableWeightKG:    chargeableWeight,
		ContainerRequirements: make([]pricing.ContainerRequirement, 0),
	}

	conditions := pricing.CalculationConditions{}

	return pricing.NewCalculationRequestWithSummary(
		input.Route,
		summary,
		conditions,
	)
}

// extractResultForLineItem: Tariff計算結果から対象LineItemの結果を抽出しRate BCの型に変換（ACL変換）
func extractResultForLineItem(
	calcResult *pricing.TariffCalculationResult,
	targetLineItemID uuid.UUID,
) (*domainrate.TariffChargeResult, error) {
	// Applied items から探す
	for _, item := range calcResult.AppliedItems {
		if item.LineItemID == targetLineItemID {
			return &domainrate.TariffChargeResult{
				LineItemID: item.LineItemID,
				ChargeCode: item.ChargeCode,
				Category:   item.Category,
				Amount:     item.Amount,
				Skipped:    false,
			}, nil
		}
	}

	// Skipped items から探す
	for _, item := range calcResult.SkippedItems {
		if item.LineItemID == targetLineItemID {
			return &domainrate.TariffChargeResult{
				LineItemID: item.LineItemID,
				ChargeCode: item.ChargeCode,
				Category:   item.Category,
				Amount:     shared.ZeroMoney("USD"),
				Skipped:    true,
				SkipReason: item.Reason,
			}, nil
		}
	}

	return nil, fmt.Errorf("line item %s not found in tariff calculation result", targetLineItemID)
}
