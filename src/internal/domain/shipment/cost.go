package shipment

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ==========================================
// 費用関連の型定義
// ==========================================

// EstimatedCost: 見積費用 (計画時点での推定費用)
// FreightEstimatorによって計算される
type EstimatedCost struct {
	TariffID      uuid.UUID
	LineItems     []CostLineItem
	TotalAmount     shared.Money
	CalculatedAt    time.Time
	CalculationBase string // "PLAN_BASED"
}

// EstimatedActualCost: 想定実費用 (トラッキング実績ベースの費用)
// CostCalculationServiceによって計算される
type EstimatedActualCost struct {
	ShipmentID      uuid.UUID
	TariffID        uuid.UUID
	SegmentCosts    []SegmentCost   // セグメント単位の内訳
	TotalAmount     shared.Money    // 合計
	CalculatedAt    time.Time
	CalculationBase string          // "TRACKING_PROGRESS"
}

// ActualCost: 実請求額 (外部から入力される確定費用)
// 物流企業からのインボイスデータ
type ActualCost struct {
	InvoiceID       uuid.UUID
	InvoiceNo       string
	ProviderID      uuid.UUID
	LineItems       []CostLineItem
	TotalAmount     shared.Money
	InvoiceDate     time.Time
	ReceivedAt      time.Time
}

// SegmentCost: セグメント単位の費用
type SegmentCost struct {
	SegmentID       uuid.UUID       // TrackingSegmentのID
	SegmentIndex    int             // セグメントの順序
	OriginLocationID uuid.UUID
	DestLocationID   uuid.UUID
	Mode            shared.TransportMode
	LineItems       []CostLineItem
	TotalAmount     shared.Money
	CalculationStatus SegmentCostStatus
	CalculatedAt    time.Time
}

// SegmentCostStatus: セグメント費用の計算ステータス
type SegmentCostStatus string

const (
	SegmentCostCompleted   SegmentCostStatus = "COMPLETED"   // 完了済み（実績ベース確定）
	SegmentCostInProgress  SegmentCostStatus = "IN_PROGRESS" // 進行中（按分計算）
	SegmentCostPlanned     SegmentCostStatus = "PLANNED"     // 未着手（計画ベース）
	SegmentCostNotApplicable SegmentCostStatus = "NOT_APPLICABLE" // 適用対象外
)

// CostLineItem: 費用明細行
// EstimatedCost, EstimatedActualCost, ActualCostで共通して使用
type CostLineItem struct {
	ID              uuid.UUID
	ChargeCode      string
	ChargeName      string
	Category        string
	Amount          shared.Money
	Quantity        *shared.Decimal // オプション：数量ベース課金の場合
	UnitPrice       *shared.Money   // オプション：単価
	AppliedScope    string          // 適用範囲の説明
	Remarks         string          // 備考
}

// ==========================================
// CostGapAnalysis: 費用差異分析
// ==========================================

// CostGapAnalysis: 見積と実績の差異分析結果
type CostGapAnalysis struct {
	ShipmentID          uuid.UUID
	AnalyzedAt          time.Time

	EstimatedTotal      shared.Money
	ActualTotal         shared.Money
	TotalGap            shared.Money        // 差額 (Actual - Estimated)
	TotalGapPercentage  float64             // 差異率 (%)

	// 項目別の差異
	ItemGaps            []CostItemGap

	// 差異の分類
	OverBudgetItems     []string            // 予算超過項目
	UnderBudgetItems    []string            // 予算未満項目
	UnexpectedCharges   []string            // 予想外の課金
	MissingCharges      []string            // 計上漏れ
}

// CostItemGap: 項目別の費用差異
type CostItemGap struct {
	ChargeCode         string
	ChargeName         string
	EstimatedAmount    shared.Money
	ActualAmount       shared.Money
	Gap                shared.Money
	GapPercentage      float64
	GapReason          string          // 差異理由
}

// ==========================================
// Methods
// ==========================================

// CalculateTotal: EstimatedCostの合計金額を計算
func (ec *EstimatedCost) CalculateTotal() shared.Money {
	total := shared.Money{
		Amount:   shared.NewDecimal(0),
		Currency: "USD", // デフォルト通貨
	}

	for _, item := range ec.LineItems {
		// 通貨が一致する場合のみ加算（簡易実装）
		if item.Amount.Currency == total.Currency {
			total.Amount = total.Amount.Add(item.Amount.Amount)
		}
	}

	return total
}

// CalculateTotal: EstimatedActualCostの合計金額を計算
func (eac *EstimatedActualCost) CalculateTotal() shared.Money {
	total := shared.Money{
		Amount:   shared.NewDecimal(0),
		Currency: "USD", // デフォルト通貨
	}

	for _, segCost := range eac.SegmentCosts {
		// 通貨が一致する場合のみ加算（簡易実装）
		if segCost.TotalAmount.Currency == total.Currency {
			total.Amount = total.Amount.Add(segCost.TotalAmount.Amount)
		}
	}

	return total
}

// CalculateTotal: ActualCostの合計金額を計算
func (ac *ActualCost) CalculateTotal() shared.Money {
	total := shared.Money{
		Amount:   shared.NewDecimal(0),
		Currency: "USD", // デフォルト通貨
	}

	for _, item := range ac.LineItems {
		// 通貨が一致する場合のみ加算（簡易実装）
		if item.Amount.Currency == total.Currency {
			total.Amount = total.Amount.Add(item.Amount.Amount)
		}
	}

	return total
}

// CalculateTotal: SegmentCostの合計金額を計算
func (sc *SegmentCost) CalculateTotal() shared.Money {
	total := shared.Money{
		Amount:   shared.NewDecimal(0),
		Currency: "USD", // デフォルト通貨
	}

	for _, item := range sc.LineItems {
		// 通貨が一致する場合のみ加算（簡易実装）
		if item.Amount.Currency == total.Currency {
			total.Amount = total.Amount.Add(item.Amount.Amount)
		}
	}

	return total
}

// IsOverBudget: 予算超過かどうかを判定
func (cg *CostGapAnalysis) IsOverBudget() bool {
	return cg.TotalGap.Amount.GreaterThan(shared.NewDecimal(0))
}

// IsWithinTolerance: 許容範囲内かどうかを判定
func (cg *CostGapAnalysis) IsWithinTolerance(tolerancePercentage float64) bool {
	absGap := cg.TotalGapPercentage
	if absGap < 0 {
		absGap = -absGap
	}
	return absGap <= tolerancePercentage
}
