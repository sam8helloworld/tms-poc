package shipment

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ShipmentCreated: 出荷案件作成イベント
type ShipmentCreated struct {
	shared.BaseEvent
}

// NewShipmentCreated: ShipmentCreatedイベントを生成
func NewShipmentCreated(shipmentID uuid.UUID) ShipmentCreated {
	return ShipmentCreated{
		BaseEvent: shared.NewBaseEvent("ShipmentCreated", shipmentID, "Shipment"),
	}
}

// ShipmentStatusChanged: 出荷案件ステータス変更イベント
type ShipmentStatusChanged struct {
	shared.BaseEvent
	OldStatus ShipmentStatus
	NewStatus ShipmentStatus
}

// NewShipmentStatusChanged: ShipmentStatusChangedイベントを生成
func NewShipmentStatusChanged(shipmentID uuid.UUID, oldStatus, newStatus ShipmentStatus) ShipmentStatusChanged {
	return ShipmentStatusChanged{
		BaseEvent: shared.NewBaseEvent("ShipmentStatusChanged", shipmentID, "Shipment"),
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}
}
