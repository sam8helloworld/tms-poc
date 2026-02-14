package route

import (
	"context"

	"github.com/google/uuid"
)

// LocationRepository: Location集約のリポジトリインターフェース
type LocationRepository interface {
	// Save: Locationを保存（新規作成または更新）
	Save(ctx context.Context, location *Location) error

	// FindByID: IDでLocationを取得
	FindByID(ctx context.Context, id LocationID) (*Location, error)

	// FindByUnLocode: UN/LOCODEでLocationを取得
	FindByUnLocode(ctx context.Context, unLocode string) (*Location, error)
}

// LaneRepository: Lane集約のリポジトリインターフェース
type LaneRepository interface {
	// Save: Laneを保存（新規作成または更新）
	Save(ctx context.Context, lane *Lane) error

	// FindByID: IDでLaneを取得
	FindByID(ctx context.Context, id LaneID) (*Lane, error)

	// FindByOriginAndDestination: 始点・終点でLaneを検索
	FindByOriginAndDestination(ctx context.Context, originID, destID uuid.UUID) ([]*Lane, error)
}

// StandardRouteRepository: StandardRoute集約のリポジトリインターフェース
type StandardRouteRepository interface {
	// Save: StandardRouteを保存（新規作成または更新）
	Save(ctx context.Context, route *StandardRoute) error

	// FindByID: IDでStandardRouteを取得
	FindByID(ctx context.Context, id StandardRouteID) (*StandardRoute, error)

	// FindActiveByShipper: 荷主のACTIVEなStandardRouteを取得
	FindActiveByShipper(ctx context.Context, shipperID uuid.UUID) ([]*StandardRoute, error)
}
