package tracking

import (
	"time"

	"github.com/google/uuid"
)

// ReconstructTrackingUnit: 永続化層からTrackingUnitを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructTrackingUnit(
	id TrackingUnitID,
	trackingNumber TrackingNumber,
	carrierID uuid.UUID,
	segments []*TrackingSegment,
	currentStatus TrackingStatus,
	lastUpdated time.Time,
) *TrackingUnit {
	return &TrackingUnit{
		ID:             id,
		TrackingNumber: trackingNumber,
		CarrierID:      carrierID,
		segments:       segments,
		currentStatus:  currentStatus,
		LastUpdated:    lastUpdated,
	}
}
