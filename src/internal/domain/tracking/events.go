package tracking

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
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
