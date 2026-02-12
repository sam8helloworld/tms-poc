package tracking

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// TrackingEventReceived: トラッキングイベント受信イベント
type TrackingEventReceived struct {
	shared.BaseEvent
	SegmentID uuid.UUID
	EventCode string
}

// NewTrackingEventReceived: TrackingEventReceivedイベントを生成
func NewTrackingEventReceived(trackingUnitID, segmentID uuid.UUID, eventCode string) TrackingEventReceived {
	return TrackingEventReceived{
		BaseEvent: shared.NewBaseEvent("TrackingEventReceived", trackingUnitID, "TrackingUnit"),
		SegmentID: segmentID,
		EventCode: eventCode,
	}
}

// TrackingRegistered: トラッキング登録イベント
type TrackingRegistered struct {
	shared.BaseEvent
	ShipmentID     uuid.UUID
	TrackingNumber string
	SegmentCount   int
}

// NewTrackingRegistered: TrackingRegisteredイベントを生成
func NewTrackingRegistered(trackingUnitID, shipmentID uuid.UUID, trackingNumber string, segmentCount int) TrackingRegistered {
	return TrackingRegistered{
		BaseEvent:      shared.NewBaseEvent("TrackingRegistered", trackingUnitID, "TrackingUnit"),
		ShipmentID:     shipmentID,
		TrackingNumber: trackingNumber,
		SegmentCount:   segmentCount,
	}
}
