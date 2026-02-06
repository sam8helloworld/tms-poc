package tracking

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ==========================================
// 1. Data Source Definition (Who updates?)
// ==========================================

type TrackingSourceType string

const (
	SourceSeaRates  TrackingSourceType = "SEARATES_API" // 海上/航空 API
	SourceManual    TrackingSourceType = "MANUAL_INPUT" // 社内オペレーター手動
	SourcePartner   TrackingSourceType = "PARTNER_EDI"  // 外部トラック業者
	SourceDriverApp TrackingSourceType = "DRIVER_APP"   // 自社配送アプリ
	SourceIoT       TrackingSourceType = "IOT_DEVICE"   // 温度ロガー等
)

// ==========================================
// Value Objects
// ==========================================

type TrackingStatus string

const (
	StatusBooked    TrackingStatus = "BOOKED"
	StatusInTransit TrackingStatus = "IN_TRANSIT"
	StatusException TrackingStatus = "EXCEPTION" // 遅延・トラブル
	StatusArrived   TrackingStatus = "ARRIVED"
)

// ==========================================
// 2. Aggregate Root: E2E Tracking
// ==========================================

type ShipmentTracking struct {
	ID         uuid.UUID
	ShipmentID uuid.UUID

	// Master Tracking Number (荷主に伝える代表番号)
	MasterTrackingNumber string

	// Segments: E2Eの工程ごとの追跡状況
	// NetworkドメインのRouteSegmentと1:1またはN:1で対応
	Segments []*TrackingSegment

	// Overall Status
	CurrentStatus TrackingStatus
	LastUpdated   time.Time
}

// UpdateSegmentStatus: 特定の区間に対して更新をかける
func (st *ShipmentTracking) UpdateSegmentStatus(
	segmentID uuid.UUID,
	event TrackingEvent,
) error {
	// 1. 対象のセグメントを探す
	var targetSeg *TrackingSegment
	for _, seg := range st.Segments {
		if seg.ID == segmentID {
			targetSeg = seg
			break
		}
	}
	if targetSeg == nil {
		return errors.New("segment not found")
	}

	// 2. セグメントの状態を更新
	targetSeg.AddEvent(event)

	// 3. 全体のステータスを再評価 (例: 全セグメント完了ならARRIVED)
	// st.recalculateOverallStatus()

	return nil
}

// ==========================================
// 3. Entity: Tracking Segment (区間管理)
// ==========================================

type TrackingSegment struct {
	ID uuid.UUID

	// 物理ルート上のどの区間か？ (Link to Network Domain)
	RouteSegmentID route.RouteSegmentID
	Mode           shared.TransportMode // OCEAN, AIR, TRUCK

	// この区間のトラッキング番号 (Masterとは別の、船社BLや航空AWB)
	CarrierTrackingNumber string

	// 誰がこの区間の情報を更新する権限を持つか？
	PrimarySource TrackingSourceType

	// 状態
	Status TrackingStatus
	Events []TrackingEvent

	// 予実 (ETA/ATA)
	EstimatedDeparture *time.Time
	ActualDeparture    *time.Time
	EstimatedArrival   *time.Time
	ActualArrival      *time.Time
}

func (ts *TrackingSegment) AddEvent(event TrackingEvent) {
	ts.Events = append(ts.Events, event)

	// 最新イベントに基づいて時刻やステータスを更新するロジック
	// if event.IsArrivalEvent() {
	// 	ts.ActualArrival = &event.Timestamp
	// 	ts.Status = StatusArrived
	// }
	// ... 他のロジック
}

// ==========================================
// 4. Value Object: Event
// ==========================================

type TrackingEvent struct {
	ID        uuid.UUID
	Timestamp time.Time
	Source    TrackingSourceType // SEARATES, MANUAL...

	Code        string // 標準化コード (e.g., "DEPT", "ARRI")
	Description string
	LocationRaw string

	// 生データ (監査用JSON)
	RawPayload string
}
