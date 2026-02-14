package shipment

import (
	"time"

	"github.com/google/uuid"
)

// ReconstructShipment: 永続化層からShipmentを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
// Note: shared.EventRecorderのeventsフィールドは復元しない（ドメインイベントテーブルから別途取得）
func ReconstructShipment(
	id uuid.UUID,
	shipmentNo string,
	shipperID, consigneeID uuid.UUID,
	status ShipmentStatus,
	plan ShipmentPlan,
	execution ShipmentExecution,
	trackingUnitIDs []uuid.UUID,
	cost ShipmentCost,
	createdAt, updatedAt time.Time,
) *Shipment {
	return &Shipment{
		ID:              id,
		ShipmentNo:      shipmentNo,
		ShipperID:       shipperID,
		ConsigneeID:     consigneeID,
		status:          status,
		Plan:            plan,
		execution:       execution,
		trackingUnitIDs: trackingUnitIDs,
		cost:            cost,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

// ReconstructShipmentExecution: 永続化層からShipmentExecutionを復元するための関数
func ReconstructShipmentExecution(milestones []Milestone) ShipmentExecution {
	return ShipmentExecution{milestones: milestones}
}
