package tracking

import (
	"context"

	"github.com/google/uuid"
)

// TrackingUnitRepository: TrackingUnit集約のリポジトリインターフェース
type TrackingUnitRepository interface {
	// Save: TrackingUnitを保存（新規作成または更新）
	Save(ctx context.Context, unit *TrackingUnit) error

	// FindByID: IDでTrackingUnitを取得
	FindByID(ctx context.Context, id uuid.UUID) (*TrackingUnit, error)

	// FindByTrackingNumber: トラッキング番号と番号種別でTrackingUnitを取得
	FindByTrackingNumber(ctx context.Context, number string, numberType TrackingNumberType) (*TrackingUnit, error)
}
