package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
	"github.com/sam8helloworld/tms-poc/internal/domain/calcparam"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/sam8helloworld/tms-poc/internal/domain/shipment"
	"github.com/sam8helloworld/tms-poc/internal/domain/tracking"
	"github.com/shopspring/decimal"
)

// ==========================================
// CostCalculationService: 費用計算ドメインサービス
// ==========================================

// CostCalculationService: 費用計算を担当するドメインサービス
// 計画ベースの見積費用と、実績ベースの想定費用を計算する
type CostCalculationService struct {
	estimator *FreightEstimator
}

// NewCostCalculationService: CostCalculationServiceの生成
func NewCostCalculationService() *CostCalculationService {
	return &CostCalculationService{
		estimator: &FreightEstimator{},
	}
}

// ==========================================
// 1. 計画時点での見積費用算出
// ==========================================

// CalculateEstimatedCost: 計画時点での見積費用を算出
// ShipmentPlanとTariffから見積費用を計算
func (ccs *CostCalculationService) CalculateEstimatedCost(
	plan *shipment.ShipmentPlan,
	tariff *commercial.Tariff,
) (shipment.EstimatedCost, error) {
	// ShipmentPlanをShipmentContextに変換
	ctx := ccs.convertPlanToContext(plan)

	// 既存のFreightEstimatorを使用して計算
	estimatedCost, err := ccs.estimator.Estimate(ctx, tariff)
	if err != nil {
		return shipment.EstimatedCost{}, fmt.Errorf("failed to calculate estimated cost: %w", err)
	}

	// 結果をshipment.EstimatedCostに変換
	lineItems := make([]shipment.CostLineItem, len(estimatedCost.LineItems))
	for i, item := range estimatedCost.LineItems {
		lineItems[i] = shipment.CostLineItem{
			ID:           uuid.New(),
			ChargeCode:   item.ChargeCode,
			ChargeName:   item.ChargeCode, // 実装では適切な名前マッピングが必要
			Category:     item.Category,
			Amount:       item.Amount,
			AppliedScope: "PLAN_BASED",
		}
	}

	result := shipment.EstimatedCost{
		RateID:          plan.RateID,
		LineItems:       lineItems,
		CalculatedAt:    time.Now(),
		CalculationBase: "PLAN_BASED",
	}
	result.TotalAmount = result.CalculateTotal()

	return result, nil
}

// ==========================================
// 2. トラッキング実績ベースの想定費用算出
// ==========================================

// CalculateEstimatedActualCost: トラッキング実績ベースの想定費用を算出
// Shipment（計画+実績）とTariffから想定実費用を計算
// セグメント単位で費用を算出する
func (ccs *CostCalculationService) CalculateEstimatedActualCost(
	ship *shipment.Shipment,
	trackingUnits []*tracking.TrackingUnit,
	tariff *commercial.Tariff,
) (shipment.EstimatedActualCost, error) {
	segmentCosts := make([]shipment.SegmentCost, 0)

	// 各TrackingUnitのセグメントごとに費用を計算
	for _, tu := range trackingUnits {
		for idx, trackingSeg := range tu.Segments() {
			segCost, err := ccs.calculateSegmentCost(
				ship,
				tu,
				trackingSeg,
				idx,
				tariff,
			)
			if err != nil {
				return shipment.EstimatedActualCost{}, fmt.Errorf(
					"failed to calculate segment cost (segment=%s): %w",
					trackingSeg.ID,
					err,
				)
			}
			segmentCosts = append(segmentCosts, segCost)
		}
	}

	result := shipment.EstimatedActualCost{
		ShipmentID:      ship.ID,
		RateID:          ship.Plan.RateID,
		SegmentCosts:    segmentCosts,
		CalculatedAt:    time.Now(),
		CalculationBase: "TRACKING_PROGRESS",
	}
	result.TotalAmount = result.CalculateTotal()

	return result, nil
}

// calculateSegmentCost: セグメント単位の費用を計算
func (ccs *CostCalculationService) calculateSegmentCost(
	ship *shipment.Shipment,
	tu *tracking.TrackingUnit,
	trackingSeg *tracking.TrackingSegment,
	segmentIndex int,
	tariff *commercial.Tariff,
) (shipment.SegmentCost, error) {
	// セグメントの状態に基づいて計算ステータスを判定
	calcStatus := ccs.determineSegmentCostStatus(trackingSeg)

	// ShipmentContextを作成（実績情報を反映）
	ctx := ccs.createContextForSegment(ship, trackingSeg)

	// Tariffの各LineItemについて、このセグメントに適用されるものを計算
	lineItems := make([]shipment.CostLineItem, 0)
	for _, tariffItem := range tariff.LineItems {
		// 適用範囲チェック
		if !tariffItem.Scope.IsApplicable(ctx) {
			continue
		}

		// 金額計算
		amount, err := tariffItem.Logic.Calculate(ctx)
		if err != nil {
			return shipment.SegmentCost{}, fmt.Errorf(
				"failed to calculate charge (code=%s): %w",
				tariffItem.ChargeCode,
				err,
			)
		}

		// 進行中の場合は按分計算を適用
		if calcStatus == shipment.SegmentCostInProgress {
			amount = ccs.applyProration(amount, trackingSeg)
		}

		lineItems = append(lineItems, shipment.CostLineItem{
			ID:           uuid.New(),
			ChargeCode:   tariffItem.ChargeCode,
			ChargeName:   tariffItem.ChargeCode,
			Category:     tariffItem.Category,
			Amount:       amount,
			AppliedScope: fmt.Sprintf("SEGMENT_%d", segmentIndex),
		})
	}

	// セグメント費用を構築
	segCost := shipment.SegmentCost{
		SegmentID:         trackingSeg.ID,
		SegmentIndex:      segmentIndex,
		OriginLocationID:  uuid.Nil, // 実装では適切に設定する必要がある
		DestLocationID:    uuid.Nil, // 実装では適切に設定する必要がある
		Mode:              trackingSeg.Mode,
		LineItems:         lineItems,
		CalculationStatus: calcStatus,
		CalculatedAt:      time.Now(),
	}
	segCost.TotalAmount = segCost.CalculateTotal()

	return segCost, nil
}

// determineSegmentCostStatus: セグメントの費用計算ステータスを判定
func (ccs *CostCalculationService) determineSegmentCostStatus(
	seg *tracking.TrackingSegment,
) shipment.SegmentCostStatus {
	switch seg.Status {
	case tracking.StatusArrived:
		return shipment.SegmentCostCompleted
	case tracking.StatusInTransit:
		return shipment.SegmentCostInProgress
	case tracking.StatusBooked:
		return shipment.SegmentCostPlanned
	case tracking.StatusException:
		return shipment.SegmentCostInProgress // 例外状態でも進行中として扱う
	default:
		return shipment.SegmentCostPlanned
	}
}

// applyProration: 進行中セグメントの按分計算
func (ccs *CostCalculationService) applyProration(
	amount shared.Money,
	seg *tracking.TrackingSegment,
) shared.Money {
	if seg.EstimatedArrival != nil && seg.ActualDeparture != nil {
		now := time.Now()
		totalDuration := seg.EstimatedArrival.Sub(*seg.ActualDeparture)
		elapsedDuration := now.Sub(*seg.ActualDeparture)

		if totalDuration > 0 {
			progressRatio := float64(elapsedDuration) / float64(totalDuration)
			if progressRatio > 1.0 {
				progressRatio = 1.0
			}
			if progressRatio < 0.0 {
				progressRatio = 0.0
			}

			return amount.Multiply(decimal.NewFromFloat(progressRatio))
		}
	}

	// デフォルト: 50%
	return amount.Multiply(decimal.NewFromFloat(0.5))
}

// ==========================================
// 3. Gap分析
// ==========================================

// AnalyzeCostGap: 想定費用と実請求額の差異を分析
func (ccs *CostCalculationService) AnalyzeCostGap(
	estimated shipment.EstimatedActualCost,
	actual shipment.ActualCost,
) (shipment.CostGapAnalysis, error) {
	// 合計差異を計算
	totalGap, _ := actual.TotalAmount.Sub(estimated.TotalAmount)

	var totalGapPercentage float64
	if !estimated.TotalAmount.IsZero() {
		totalGapPercentage, _ = totalGap.Amount.Div(estimated.TotalAmount.Amount).Mul(decimal.NewFromInt(100)).Float64()
	}

	// 項目別の差異を計算
	itemGaps := ccs.calculateItemGaps(estimated, actual)

	// 差異分類
	overBudget, underBudget, unexpected, missing := ccs.classifyGaps(itemGaps)

	return shipment.CostGapAnalysis{
		ShipmentID:         estimated.ShipmentID,
		AnalyzedAt:         time.Now(),
		EstimatedTotal:     estimated.TotalAmount,
		ActualTotal:        actual.TotalAmount,
		TotalGap:           totalGap,
		TotalGapPercentage: totalGapPercentage,
		ItemGaps:           itemGaps,
		OverBudgetItems:    overBudget,
		UnderBudgetItems:   underBudget,
		UnexpectedCharges:  unexpected,
		MissingCharges:     missing,
	}, nil
}

// calculateItemGaps: 項目別の差異を計算
func (ccs *CostCalculationService) calculateItemGaps(
	estimated shipment.EstimatedActualCost,
	actual shipment.ActualCost,
) []shipment.CostItemGap {
	// EstimatedとActualの項目をマッピング
	estimatedMap := make(map[string]shared.Money)
	for _, segCost := range estimated.SegmentCosts {
		for _, item := range segCost.LineItems {
			if existing, ok := estimatedMap[item.ChargeCode]; ok {
				result, err := existing.Add(item.Amount)
				if err == nil {
					estimatedMap[item.ChargeCode] = result
				}
			} else {
				estimatedMap[item.ChargeCode] = item.Amount
			}
		}
	}

	actualMap := make(map[string]shared.Money)
	for _, item := range actual.LineItems {
		if existing, ok := actualMap[item.ChargeCode]; ok {
			result, err := existing.Add(item.Amount)
			if err == nil {
				actualMap[item.ChargeCode] = result
			}
		} else {
			actualMap[item.ChargeCode] = item.Amount
		}
	}

	// 全ChargeCodeを収集
	allCodes := make(map[string]bool)
	for code := range estimatedMap {
		allCodes[code] = true
	}
	for code := range actualMap {
		allCodes[code] = true
	}

	// 各項目の差異を計算
	gaps := make([]shipment.CostItemGap, 0, len(allCodes))
	for code := range allCodes {
		est := estimatedMap[code]
		act := actualMap[code]

		gap, _ := act.Sub(est)

		var gapPercentage float64
		if !est.IsZero() {
			gapPercentage, _ = gap.Amount.Div(est.Amount).Mul(decimal.NewFromInt(100)).Float64()
		}

		gaps = append(gaps, shipment.CostItemGap{
			ChargeCode:      code,
			ChargeName:      code,
			EstimatedAmount: est,
			ActualAmount:    act,
			Gap:             gap,
			GapPercentage:   gapPercentage,
			GapReason:       ccs.deriveGapReason(code, gap, gapPercentage),
		})
	}

	return gaps
}

// classifyGaps: 差異を分類
func (ccs *CostCalculationService) classifyGaps(
	gaps []shipment.CostItemGap,
) (overBudget, underBudget, unexpected, missing []string) {
	for _, gap := range gaps {
		if gap.EstimatedAmount.IsZero() && !gap.ActualAmount.IsZero() {
			unexpected = append(unexpected, gap.ChargeCode)
		} else if !gap.EstimatedAmount.IsZero() && gap.ActualAmount.IsZero() {
			missing = append(missing, gap.ChargeCode)
		} else if gap.Gap.IsPositive() {
			overBudget = append(overBudget, gap.ChargeCode)
		} else if gap.Gap.IsNegative() {
			underBudget = append(underBudget, gap.ChargeCode)
		}
	}
	return
}

// deriveGapReason: 差異理由を推測
func (ccs *CostCalculationService) deriveGapReason(
	code string,
	gap shared.Money,
	gapPercentage float64,
) string {
	if gap.IsZero() {
		return "No gap"
	}

	absGap := gapPercentage
	if absGap < 0 {
		absGap = -absGap
	}

	if absGap < 5 {
		return "Within tolerance"
	} else if absGap < 15 {
		return "Minor variance"
	} else {
		return "Significant variance - requires investigation"
	}
}

// ==========================================
// Helper Methods
// ==========================================

// convertPlanToContext: ShipmentPlanをShipmentContextに変換
func (ccs *CostCalculationService) convertPlanToContext(
	plan *shipment.ShipmentPlan,
) calcparam.ShipmentContext {
	return calcparam.ShipmentContext{
		Route:      plan.PlannedRoute,
		Quantity:   plan.GetTotalQuantity(),
		WeightKG:   plan.GetTotalWeight(),
		VolumeM3:   plan.GetTotalVolume(),
		Attributes: plan.TransportRequirements,
	}
}

// createContextForSegment: セグメント単位のShipmentContextを作成
func (ccs *CostCalculationService) createContextForSegment(
	ship *shipment.Shipment,
	trackingSeg *tracking.TrackingSegment,
) calcparam.ShipmentContext {
	return calcparam.ShipmentContext{
		Route:      ship.Plan.PlannedRoute,
		Quantity:   ship.Plan.GetTotalQuantity(),
		WeightKG:   ship.Plan.GetTotalWeight(),
		VolumeM3:   ship.Plan.GetTotalVolume(),
		Attributes: ship.Plan.TransportRequirements,
	}
}
