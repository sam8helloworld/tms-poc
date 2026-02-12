package shipment

import (
	"context"

	"github.com/google/uuid"
)

// ShipmentRepository: Shipment集約のリポジトリインターフェース
type ShipmentRepository interface {
	// Save: Shipmentを保存（新規作成または更新）
	Save(ctx context.Context, shipment *Shipment) error

	// FindByID: IDでShipmentを取得
	FindByID(ctx context.Context, id uuid.UUID) (*Shipment, error)
}
