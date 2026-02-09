package route

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ==========================================
// Aggregate Root: StandardRoute (標準ルート)
// ==========================================

// StandardRouteID: 標準ルートの識別子
type StandardRouteID uuid.UUID

// StandardRouteStatus: 標準ルートのステータス
type StandardRouteStatus string

const (
	StandardRouteStatusActive   StandardRouteStatus = "ACTIVE"
	StandardRouteStatusArchived StandardRouteStatus = "ARCHIVED"
)

// StandardRoute: 荷主が管理する「標準ルート」(Aggregate Root)
// 入札（RFQ）、予算策定、発注時の指定、リードタイム基準値の設定に使用する
// 性質: 「規範（Prescriptive）」 - あるべき姿を定義するマスタデータ
type StandardRoute struct {
	shared.EventRecorder

	ID                    StandardRouteID
	Name                  string
	ShipperID             uuid.UUID
	OriginLocationID      LocationID
	DestinationLocationID LocationID

	// 計画用の区間定義（private）
	legs []StandardRouteLeg

	// ステータス（private）
	status StandardRouteStatus

	// KPI基準値
	StandardLeadTimeDays int
	TargetCost           *shared.Money

	// 有効期間
	ValidPeriod shared.DateRange

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewStandardRoute: StandardRouteのファクトリ関数
func NewStandardRoute(
	name string,
	shipperID uuid.UUID,
	originLocationID LocationID,
	destinationLocationID LocationID,
	legs []StandardRouteLeg,
	standardLeadTimeDays int,
	targetCost *shared.Money,
	validPeriod shared.DateRange,
) (*StandardRoute, error) {
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "name is required")
	}
	if shipperID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipperID is required")
	}
	if len(legs) == 0 {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "at least one leg is required")
	}
	if standardLeadTimeDays <= 0 {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "standardLeadTimeDays must be positive")
	}

	now := time.Now()
	id := StandardRouteID(uuid.New())

	// レグのコピーを作成
	legsCopy := make([]StandardRouteLeg, len(legs))
	copy(legsCopy, legs)

	sr := &StandardRoute{
		ID:                    id,
		Name:                  name,
		ShipperID:             shipperID,
		OriginLocationID:      originLocationID,
		DestinationLocationID: destinationLocationID,
		legs:                  legsCopy,
		status:                StandardRouteStatusActive,
		StandardLeadTimeDays:  standardLeadTimeDays,
		TargetCost:            targetCost,
		ValidPeriod:           validPeriod,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	sr.RecordEvent(NewStandardRouteCreated(uuid.UUID(id)))
	return sr, nil
}

// Status: ステータスのgetter
func (sr *StandardRoute) Status() StandardRouteStatus {
	return sr.status
}

// Legs: レグのgetter（コピーを返却）
func (sr *StandardRoute) Legs() []StandardRouteLeg {
	result := make([]StandardRouteLeg, len(sr.legs))
	copy(result, sr.legs)
	return result
}

// Archive: 標準ルートをアーカイブする
func (sr *StandardRoute) Archive() error {
	if sr.status == StandardRouteStatusArchived {
		return shared.NewDomainError(shared.ErrInvalidState, "standard route is already archived")
	}
	sr.status = StandardRouteStatusArchived
	sr.UpdatedAt = time.Now()
	sr.RecordEvent(NewStandardRouteArchived(uuid.UUID(sr.ID)))
	return nil
}

// ==========================================
// Value Object: StandardRouteLeg (標準ルート区間)
// ==========================================

// StandardRouteLeg: 標準ルートの1区間を表す値オブジェクト
// 計画コンテキストに特化した属性を持つ（目標モード、目標日数）
// PhysicalRoute の RouteSegment（実データ向け）とは異なる関心事を持つ
type StandardRouteLeg struct {
	SequenceOrder       int
	OriginLocationID    LocationID
	DestLocationID      LocationID
	TargetMode          shared.TransportMode
	StandardTransitDays int    // この区間の目標所要日数
	MasterLaneID        *LaneID // マスタのLaneへの参照（任意）
}
