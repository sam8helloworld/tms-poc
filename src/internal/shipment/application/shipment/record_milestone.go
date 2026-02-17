package shipment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/shipment/domain/shipment"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// RecordMilestoneInput: マイルストーン記録の入力
type RecordMilestoneInput struct {
	ShipmentID       uuid.UUID
	MilestoneType    domain.MilestoneType
	OccurredAt       time.Time
	SourceDocumentID uuid.UUID
	SourceDocType    shared.DocType
	Payload          domain.MilestonePayload
}

// RecordMilestoneOutput: マイルストーン記録の出力
type RecordMilestoneOutput struct {
	ShipmentID    uuid.UUID
	MilestoneType domain.MilestoneType
	NewStatus     string
}

// RecordMilestoneUseCase: マイルストーン記録ユースケース
type RecordMilestoneUseCase struct {
	shipmentRepo domain.ShipmentRepository
}

// NewRecordMilestoneUseCase: コンストラクタ
func NewRecordMilestoneUseCase(
	shipmentRepo domain.ShipmentRepository,
) *RecordMilestoneUseCase {
	return &RecordMilestoneUseCase{
		shipmentRepo: shipmentRepo,
	}
}

// Execute: ユースケースを実行
func (uc *RecordMilestoneUseCase) Execute(
	ctx context.Context,
	input RecordMilestoneInput,
) (*RecordMilestoneOutput, error) {
	ship, err := uc.shipmentRepo.FindByID(ctx, input.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("find shipment: %w", err)
	}
	if ship == nil {
		return nil, shared.NewDomainError(shared.ErrNotFound,
			fmt.Sprintf("shipment not found: %s", input.ShipmentID))
	}

	if err := ship.RecordMilestone(
		input.MilestoneType,
		input.OccurredAt,
		input.SourceDocumentID,
		input.SourceDocType,
		input.Payload,
	); err != nil {
		return nil, fmt.Errorf("record milestone: %w", err)
	}

	if err := uc.shipmentRepo.Save(ctx, ship); err != nil {
		return nil, fmt.Errorf("save shipment: %w", err)
	}

	return &RecordMilestoneOutput{
		ShipmentID:    ship.ID,
		MilestoneType: input.MilestoneType,
		NewStatus:     string(ship.Status()),
	}, nil
}
