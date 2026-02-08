package shipment

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/shopspring/decimal"
)

// ==========================================
// Aggregate Root: Shipment (出荷案件)
// ==========================================

type ShipmentID = uuid.UUID

// Shipment: 出荷案件 (Aggregate Root)
// 荷主視点での「1つの仕事」を表す
// 商流コンテキストの主役
type Shipment struct {
	shared.EventRecorder

	ID          ShipmentID
	ShipmentNo  string // 出荷番号 (荷主が参照する番号)
	ShipperID   uuid.UUID
	ConsigneeID uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// 計画情報
	Plan ShipmentPlan

	// 追跡単位への参照 (IDのみ)
	// 1つのShipmentが複数のコンテナ(TrackingUnit)に分かれる場合がある
	trackingUnitIDs []uuid.UUID

	// 費用情報
	cost ShipmentCost

	// 要約ステータス (Derived Status)
	// 全TrackingUnitの状態から導出される
	status ShipmentStatus
}

// NewShipment: Shipmentのファクトリ関数
func NewShipment(
	shipmentNo string,
	shipperID uuid.UUID,
	consigneeID uuid.UUID,
	plan ShipmentPlan,
) (*Shipment, error) {
	if shipmentNo == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipmentNo is required")
	}
	if shipperID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipperID is required")
	}
	if consigneeID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "consigneeID is required")
	}

	now := time.Now()
	id := uuid.New()
	s := &Shipment{
		ID:              id,
		ShipmentNo:      shipmentNo,
		ShipperID:       shipperID,
		ConsigneeID:     consigneeID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Plan:            plan,
		trackingUnitIDs: make([]uuid.UUID, 0),
		status:          StatusPlanned,
	}
	s.RecordEvent(NewShipmentCreated(id))
	return s, nil
}

// Status: ステータスのgetter
func (s *Shipment) Status() ShipmentStatus {
	return s.status
}

// Cost: 費用情報のgetter
func (s *Shipment) Cost() ShipmentCost {
	return s.cost
}

// TrackingUnitIDs: TrackingUnit IDsのgetter（コピーを返却）
func (s *Shipment) TrackingUnitIDs() []uuid.UUID {
	result := make([]uuid.UUID, len(s.trackingUnitIDs))
	copy(result, s.trackingUnitIDs)
	return result
}

// ==========================================
// Entities: ShipmentPlan (計画情報)
// ==========================================

// ShipmentPlan: 出荷計画情報 (Entity)
// 予定ルート、貨物情報、使用する料金表を保持
type ShipmentPlan struct {
	// 基準とした標準ルートへの参照（任意）
	// StandardRouteのコピー時点の内容をPlannedRouteに保持し、IDで追跡可能にする
	StandardRouteID *uuid.UUID

	// 物理ルート（StandardRouteからコピーされた計画値）
	PlannedRoute route.PhysicalRoute

	// 貨物明細
	Items []ShipmentItem

	// 使用するレート（社内レート）
	RateID uuid.UUID

	// 輸送要求事項
	TransportRequirements map[string]interface{} // 温度管理、危険物等の属性
}

// GetTotalWeight: 全貨物の合計重量を計算
func (sp *ShipmentPlan) GetTotalWeight() decimal.Decimal {
	total := decimal.Zero
	for _, item := range sp.Items {
		total = total.Add(item.WeightKG)
	}
	return total
}

// GetTotalVolume: 全貨物の合計容積を計算
func (sp *ShipmentPlan) GetTotalVolume() decimal.Decimal {
	total := decimal.Zero
	for _, item := range sp.Items {
		total = total.Add(item.VolumeM3)
	}
	return total
}

// GetTotalQuantity: 全貨物の合計数量を計算
func (sp *ShipmentPlan) GetTotalQuantity() decimal.Decimal {
	total := decimal.Zero
	for _, item := range sp.Items {
		total = total.Add(item.Quantity)
	}
	return total
}

// ==========================================
// Entities: ShipmentItem (貨物明細)
// ==========================================
type ShipmentItemID = uuid.UUID

// ShipmentItem: 貨物明細 (Entity)
// 例: "T-shirts 500箱", "Jeans 300箱"
type ShipmentItem struct {
	ID          ShipmentItemID
	Commodity   string          // 商品名
	HSCode      string          // HSコード (関税分類)
	Quantity    decimal.Decimal // 数量
	WeightKG    decimal.Decimal // 重量 (kg)
	VolumeM3    decimal.Decimal // 容積 (m³)
	PackageType string          // 梱包形態 (Carton, Pallet, etc.)

	// このアイテムがどのTrackingUnitに積まれているか
	// (分割出荷の場合に重要)
	LoadedOnTrackingID *uuid.UUID

	// カスタム属性
	Attributes map[string]interface{}
}

// ==========================================
// Entities: ShipmentCost (費用情報)
// ==========================================

// ShipmentCost: 費用管理 (Entity)
// 見積、想定実費用、実請求額を管理
type ShipmentCost struct {
	// 見積費用 (計画時点での推定費用)
	EstimatedCost *EstimatedCost

	// 想定実費用 (トラッキング実績ベースの費用)
	EstimatedActualCost *EstimatedActualCost

	// 実請求額 (外部から入力される確定費用)
	ActualCost *ActualCost

	// 費用確定フラグ
	IsFinalized bool
	FinalizedAt *time.Time
}

// ==========================================
// Methods: Shipment (集約ルート)
// ==========================================

// AddTrackingUnitID: TrackingUnit IDを追加
func (s *Shipment) AddTrackingUnitID(trackingUnitID uuid.UUID) {
	// 重複チェック
	for _, id := range s.trackingUnitIDs {
		if id == trackingUnitID {
			return
		}
	}
	s.trackingUnitIDs = append(s.trackingUnitIDs, trackingUnitID)
	s.UpdatedAt = time.Now()
}

// UpdateShipmentStatus: ステータスを更新（ドメインサービスから呼ばれる）
func (s *Shipment) UpdateShipmentStatus(newStatus ShipmentStatus) {
	if s.status != newStatus {
		oldStatus := s.status
		s.status = newStatus
		s.UpdatedAt = time.Now()
		s.RecordEvent(NewShipmentStatusChanged(s.ID, oldStatus, newStatus))
	}
}

// SetEstimatedCost: 見積費用を設定
func (s *Shipment) SetEstimatedCost(c EstimatedCost) {
	s.cost.EstimatedCost = &c
	s.UpdatedAt = time.Now()
}

// SetEstimatedActualCost: 想定実費用を設定
func (s *Shipment) SetEstimatedActualCost(c EstimatedActualCost) {
	s.cost.EstimatedActualCost = &c
	s.UpdatedAt = time.Now()
}

// SetActualCost: 実請求額を設定して確定
func (s *Shipment) SetActualCost(c ActualCost) {
	s.cost.ActualCost = &c
	s.cost.IsFinalized = true
	now := time.Now()
	s.cost.FinalizedAt = &now
	s.UpdatedAt = now
}
