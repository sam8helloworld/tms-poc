package tracking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// RegisterShipmentTrackingUseCase: Shipmentに対するトラッキング登録ユースケース
// TrackingUnitを作成し、TrackingRegisteredイベントを発行する
// Shipmentへの紐づけはShipmentコンテキスト側のイベントハンドラが行う（結果整合性）
type RegisterShipmentTrackingUseCase struct {
	trackingRepo   domain.TrackingUnitRepository
	eventPublisher shared.DomainEventPublisher
}

// NewRegisterShipmentTrackingUseCase: コンストラクタ
func NewRegisterShipmentTrackingUseCase(
	trackingRepo domain.TrackingUnitRepository,
	eventPublisher shared.DomainEventPublisher,
) *RegisterShipmentTrackingUseCase {
	return &RegisterShipmentTrackingUseCase{
		trackingRepo:   trackingRepo,
		eventPublisher: eventPublisher,
	}
}

// Execute: ユースケースを実行
func (uc *RegisterShipmentTrackingUseCase) Execute(
	ctx context.Context,
	input RegisterShipmentTrackingInput,
) (*RegisterShipmentTrackingOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewRegisterTrackingError("INVALID_INPUT", err.Error())
	}

	// 2. TrackingSegments構築
	segments := make([]*domain.TrackingSegment, 0, len(input.Segments))
	for _, seg := range input.Segments {
		segments = append(segments, &domain.TrackingSegment{
			ID:                     uuid.New(),
			ActualOriginLocationID: seg.ActualOriginLocationID,
			ActualDestLocationID:   seg.ActualDestLocationID,
			Mode:                   seg.Mode,
			CarrierTrackingNumber:  seg.CarrierTrackingNumber,
			PrimarySource:          seg.PrimarySource,
			Status:                 domain.StatusBooked,
			Events:                 make([]domain.TrackingEvent, 0),
		})
	}

	// 3. TrackingUnit生成
	trackingNumber := domain.TrackingNumber{
		Number:             input.TrackingNumber,
		TrackingNumberType: input.TrackingNumberType,
	}
	unit, err := domain.NewTrackingUnit(trackingNumber, input.CarrierID, segments)
	if err != nil {
		return nil, NewRegisterTrackingError("TRACKING_UNIT_CREATE_ERROR", err.Error())
	}

	// 4. TrackingRegisteredイベント記録
	unit.RecordEvent(domain.NewTrackingRegistered(
		uuid.UUID(unit.ID),
		input.ShipmentID,
		input.TrackingNumber,
		len(segments),
	))

	// 5. TrackingUnit永続化
	if err := uc.trackingRepo.Save(ctx, unit); err != nil {
		return nil, NewRegisterTrackingError("TRACKING_SAVE_ERROR", "failed to save tracking unit").
			WithCause(err)
	}

	// 6. ドメインイベント発行（Shipmentコンテキストがこれを購読してtrackingUnitIDを追加する）
	if err := uc.eventPublisher.Publish(ctx, unit.PullEvents()); err != nil {
		return nil, NewRegisterTrackingError("EVENT_PUBLISH_ERROR", "failed to publish domain events").
			WithCause(err)
	}

	// 7. 出力DTOの作成
	output := &RegisterShipmentTrackingOutput{
		TrackingUnitID: uuid.UUID(unit.ID),
		ShipmentID:     input.ShipmentID,
		TrackingNumber: input.TrackingNumber,
		SegmentCount:   len(segments),
		Status:         unit.CurrentStatus(),
		CreatedAt:      time.Now(),
	}

	return output, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *RegisterShipmentTrackingUseCase) validateInput(input RegisterShipmentTrackingInput) error {
	if input.ShipmentID == uuid.Nil {
		return errors.New("shipment ID is required")
	}
	if input.TrackingNumber == "" {
		return errors.New("tracking number is required")
	}
	if input.TrackingNumberType == "" {
		return errors.New("tracking number type is required")
	}
	if input.CarrierID == uuid.Nil {
		return errors.New("carrier ID is required")
	}
	if len(input.Segments) == 0 {
		return errors.New("at least one segment is required")
	}
	for i, seg := range input.Segments {
		if seg.ActualOriginLocationID == uuid.Nil {
			return fmt.Errorf("segment[%d]: origin location ID is required", i)
		}
		if seg.ActualDestLocationID == uuid.Nil {
			return fmt.Errorf("segment[%d]: destination location ID is required", i)
		}
		if seg.Mode == "" {
			return fmt.Errorf("segment[%d]: transport mode is required", i)
		}
		if seg.PrimarySource == "" {
			return fmt.Errorf("segment[%d]: primary source is required", i)
		}
	}
	return nil
}
