package shipment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	domain "github.com/sam8helloworld/tms-poc/internal/shipment/domain/shipment"
)

// CreateShipmentInput: 輸送案件作成の入力
type CreateShipmentInput struct {
	ShipmentNo         string
	ShipperID          uuid.UUID
	ConsigneeID        uuid.UUID
	OriginLocationID   route.LocationID
	DestLocationID     route.LocationID
	Items              []domain.ShipmentItem
	RateID             uuid.UUID
	StandardRouteID    *uuid.UUID
}

// CreateShipmentOutput: 輸送案件作成の出力
type CreateShipmentOutput struct {
	ShipmentID uuid.UUID
	ShipmentNo string
	Status     string
	CreatedAt  time.Time
}

// CreateShipmentUseCase: 輸送案件作成ユースケース
type CreateShipmentUseCase struct {
	shipmentRepo domain.ShipmentRepository
}

// NewCreateShipmentUseCase: コンストラクタ
func NewCreateShipmentUseCase(
	shipmentRepo domain.ShipmentRepository,
) *CreateShipmentUseCase {
	return &CreateShipmentUseCase{
		shipmentRepo: shipmentRepo,
	}
}

// Execute: ユースケースを実行
func (uc *CreateShipmentUseCase) Execute(
	ctx context.Context,
	input CreateShipmentInput,
) (*CreateShipmentOutput, error) {
	plan := domain.ShipmentPlan{
		StandardRouteID: input.StandardRouteID,
		PlannedRoute: route.PhysicalRoute{
			OriginID:      input.OriginLocationID,
			DestinationID: input.DestLocationID,
		},
		Items:  input.Items,
		RateID: input.RateID,
	}

	ship, err := domain.NewShipment(input.ShipmentNo, input.ShipperID, input.ConsigneeID, plan)
	if err != nil {
		return nil, fmt.Errorf("create shipment: %w", err)
	}

	if err := uc.shipmentRepo.Save(ctx, ship); err != nil {
		return nil, fmt.Errorf("save shipment: %w", err)
	}

	return &CreateShipmentOutput{
		ShipmentID: ship.ID,
		ShipmentNo: ship.ShipmentNo,
		Status:     string(ship.Status()),
		CreatedAt:  ship.CreatedAt,
	}, nil
}
