package shipment

import (
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// ==========================================
// Domain Service: ShipmentStatusUpdater
// ==========================================

// ShipmentStatusUpdater: Shipmentのステータスを更新するドメインサービス
// TrackingUnit集約の状態を取得し、Shipmentのステータスを計算・更新する
type ShipmentStatusUpdater struct{}

// NewShipmentStatusUpdater: コンストラクタ
func NewShipmentStatusUpdater() *ShipmentStatusUpdater {
	return &ShipmentStatusUpdater{}
}

// UpdateStatus: 全TrackingUnitの状態からShipmentの要約ステータスを更新
func (u *ShipmentStatusUpdater) UpdateStatus(
	ship *Shipment,
	trackingUnits []*tracking.TrackingUnit,
) {
	if len(trackingUnits) == 0 {
		ship.UpdateShipmentStatus(StatusPlanned)
		return
	}

	allCompleted := true
	anyInTransit := false
	anyException := false

	for _, tu := range trackingUnits {
		switch tu.CurrentStatus() {
		case shared.StatusInTransit:
			anyInTransit = true
			allCompleted = false
		case shared.StatusException:
			anyException = true
			allCompleted = false
		case shared.StatusBooked:
			allCompleted = false
		case shared.StatusArrived:
			// 到着済み - 継続チェック
		default:
			allCompleted = false
		}
	}

	var newStatus ShipmentStatus
	if allCompleted {
		newStatus = StatusCompleted
	} else if anyException {
		newStatus = StatusException
	} else if anyInTransit {
		newStatus = StatusInTransit
	} else {
		newStatus = StatusBooked
	}

	ship.UpdateShipmentStatus(newStatus)
}
