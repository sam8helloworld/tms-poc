package tracking

import (
	"errors"
	"time"

	"github.com/google/uuid"
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

// TrackingStatus: shared.TrackingStatus を再エクスポート
// 互換性のために残す
type TrackingStatus = shared.TrackingStatus

const (
	StatusBooked    = shared.StatusBooked
	StatusInTransit = shared.StatusInTransit
	StatusException = shared.StatusException
	StatusArrived   = shared.StatusArrived
)

// ==========================================
// 2. Aggregate Root: TrackingUnit (追跡単位)
// ==========================================

type TrackingUnitID uuid.UUID
type TrackingNumberType string

const (
	TrackingNumberContainer TrackingNumberType = "CONTAINER"
	TrackingNumberAWB       TrackingNumberType = "AIRWAY_BILL"
	TrackingNumberBL        TrackingNumberType = "BILL_OF_LADING"
	TrackingNumberBookNo    TrackingNumberType = "BOOKING_NUMBER"
)

type TrackingNumber struct {
	Number             string
	TrackingNumberType TrackingNumberType
}

// TrackingUnit: 追跡の最小単位 (Aggregate Root)
// 物理的な輸送単位（コンテナ1本、トラック1台など）を表す
// これがSeaRates API等の更新対象になる
// 旧 ShipmentTracking からリネーム
type TrackingUnit struct {
	ID TrackingUnitID

	// 物理的な識別子
	TrackingNumber TrackingNumber // Container No, AWB, Master B/L No
	CarrierID      uuid.UUID

	// Segments: E2Eの工程ごとの追跡状況
	// NetworkドメインのRouteSegmentと1:1またはN:1で対応
	Segments []*TrackingSegment

	// Overall Status (実行ステータス - Source of Truth)
	CurrentStatus TrackingStatus
	LastUpdated   time.Time
}

// UpdateSegmentStatus: 特定の区間に対して更新をかける
func (tu *TrackingUnit) UpdateSegmentStatus(
	segmentID uuid.UUID,
	event TrackingEvent,
) error {
	// 1. 対象のセグメントを探す
	var targetSeg *TrackingSegment
	for _, seg := range tu.Segments {
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
	tu.recalculateOverallStatus()

	return nil
}

// recalculateOverallStatus: 全セグメントの状態から全体ステータスを再計算
func (tu *TrackingUnit) recalculateOverallStatus() {
	if len(tu.Segments) == 0 {
		tu.CurrentStatus = StatusBooked
		return
	}

	allArrived := true
	anyInTransit := false
	anyException := false

	for _, seg := range tu.Segments {
		switch seg.Status {
		case StatusInTransit:
			anyInTransit = true
			allArrived = false
		case StatusException:
			anyException = true
			allArrived = false
		case StatusBooked:
			allArrived = false
		case StatusArrived:
			// 到着済み - 継続チェック
		default:
			allArrived = false
		}
	}

	if allArrived {
		tu.CurrentStatus = StatusArrived
	} else if anyException {
		tu.CurrentStatus = StatusException
	} else if anyInTransit {
		tu.CurrentStatus = StatusInTransit
	} else {
		tu.CurrentStatus = StatusBooked
	}

	tu.LastUpdated = time.Now()
}

// ==========================================
// 3. Entity: Tracking Segment (区間管理)
// ==========================================

type TrackingSegment struct {
	ID uuid.UUID

	// 実際の発着地（実績として記録された場所）
	ActualOriginLocationID uuid.UUID
	ActualDestLocationID   uuid.UUID
	Mode                   shared.TransportMode // OCEAN, AIR, TRUCK

	// この区間のトラッキング番号 (Masterとは別の、船社BLや航空AWB)
	CarrierTrackingNumber string

	// 誰がこの区間の情報を更新する権限を持つか？
	PrimarySource TrackingSourceType

	// 状態
	Status TrackingStatus
	Events []TrackingEvent

	// 実績時刻（計画との対応はShipmentで管理）
	ActualDeparture *time.Time
	ActualArrival   *time.Time

	// ETAは外部システム（キャリア）からの予測値
	EstimatedArrival *time.Time
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
