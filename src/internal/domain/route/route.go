package route

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/shopspring/decimal"
)

// PhysicalRoute: 順序を持った区間の集合体
type PhysicalRoute struct {
	ID            uuid.UUID
	OriginID      uuid.UUID
	DestinationID uuid.UUID

	// StopsとLegsを分けるのではなく、
	// 「移動の1単位(Segment)」のリストとして順序を保証する
	Segments []RouteSegment
}

// RouteSegment: 「A地点からB地点への移動」を表す最小単位
// 配列のインデックス 0, 1, 2... がそのまま輸送順序になる
type RouteSegment struct {
	SequenceOrder int // 念のための順序番号 (DB保存用)

	// 出発地情報 (Contextual Stop)
	OriginLocationID uuid.UUID
	OriginType       shared.LocationType // POL, WAREHOUSE

	// 到着地情報
	DestLocationID uuid.UUID
	DestType       shared.LocationType // POD, WAREHOUSE

	// 移動手段 (Contextual Leg)
	Mode       shared.TransportMode
	DistanceKm decimal.Decimal

	// マスタのLaneIDへの参照 (もしマスタが存在する場合)
	MasterLaneID *uuid.UUID
}
