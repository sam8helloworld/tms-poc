package sop

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// SOPDefinitionRepository: SOPDefinition集約のリポジトリインターフェース
type SOPDefinitionRepository interface {
	// Save: SOPDefinitionを保存（新規作成または更新）
	Save(ctx context.Context, def *SOPDefinition) error

	// FindByID: IDでSOPDefinitionを取得
	FindByID(ctx context.Context, id uuid.UUID) (*SOPDefinition, error)

	// FindActiveByScope: 適用条件に合致するACTIVEなSOPDefinitionを検索
	FindActiveByScope(ctx context.Context, direction shared.TradeDirection, mode shared.TransportMode, originCountry, destCountry *string) ([]*SOPDefinition, error)
}

// SOPInstanceRepository: SOPInstance集約のリポジトリインターフェース
type SOPInstanceRepository interface {
	// Save: SOPInstanceを保存（新規作成または更新）
	Save(ctx context.Context, instance *SOPInstance) error

	// FindByID: IDでSOPInstanceを取得
	FindByID(ctx context.Context, id uuid.UUID) (*SOPInstance, error)

	// FindByShipmentID: ShipmentIDでSOPInstanceを取得
	FindByShipmentID(ctx context.Context, shipmentID uuid.UUID) (*SOPInstance, error)
}
