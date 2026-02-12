package shipment

import (
	"context"
	"fmt"

	"github.com/sam8helloworld/tms-poc/internal/shared"
	domain "github.com/sam8helloworld/tms-poc/internal/shipment/domain/shipment"
	tracking "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// TrackingRegisteredHandler: TrackingRegisteredイベントを購読し、ShipmentにTrackingUnitIDを紐づけるハンドラ
// Trackingコンテキストが発行したイベントをShipmentコンテキスト側で受信し、
// 自コンテキストのリポジトリを使って更新する（結果整合性）
type TrackingRegisteredHandler struct {
	shipmentRepo domain.ShipmentRepository
}

// NewTrackingRegisteredHandler: コンストラクタ
func NewTrackingRegisteredHandler(
	shipmentRepo domain.ShipmentRepository,
) *TrackingRegisteredHandler {
	return &TrackingRegisteredHandler{
		shipmentRepo: shipmentRepo,
	}
}

// Handle: TrackingRegisteredイベントを処理する
func (h *TrackingRegisteredHandler) Handle(ctx context.Context, event tracking.TrackingRegistered) error {
	// 1. 対象のShipmentを取得
	ship, err := h.shipmentRepo.FindByID(ctx, event.ShipmentID)
	if err != nil {
		return fmt.Errorf("failed to fetch shipment: %w", err)
	}
	if ship == nil {
		return shared.NewDomainError(shared.ErrNotFound,
			fmt.Sprintf("shipment not found: %s", event.ShipmentID))
	}

	// 2. TrackingUnitIDを追加（Shipment集約内のロジックで重複チェック）
	trackingUnitID := event.AggregateID()
	ship.AddTrackingUnitID(trackingUnitID)

	// 3. 保存
	if err := h.shipmentRepo.Save(ctx, ship); err != nil {
		return fmt.Errorf("failed to save shipment: %w", err)
	}

	return nil
}
