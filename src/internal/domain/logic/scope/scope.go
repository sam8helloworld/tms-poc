package scope

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/context"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ServiceScope: この料金が適用される「物流サービスの範囲」を定義するインターフェース
type ServiceScope interface {
	IsApplicable(ctx context.ShipmentContext) bool
}

// LocationService: 場所に対するサービス (THC, 保管, 通関)
type LocationService struct {
	LocationID  uuid.UUID
	ServiceType string // HANDLING, STORAGE
}

func (s LocationService) IsApplicable(ctx context.ShipmentContext) bool {
	// 全セグメントを走査し、この場所が「出発地」か「到着地」として登場するか確認
	for _, seg := range ctx.Route.Segments {
		if seg.OriginLocationID == s.LocationID {
			return true
		}
		if seg.DestLocationID == s.LocationID {
			return true
		}
	}
	return false
}

// TransportationService: 移動に対するサービス (海上運賃, ドレージ)
type TransportationService struct {
	OriginID      uuid.UUID
	DestinationID uuid.UUID
	Mode          shared.TransportMode
}

func (s TransportationService) IsApplicable(ctx context.ShipmentContext) bool {
	// 全セグメントを走査し、指定された「区間移動」と一致するものがあるか確認
	for _, seg := range ctx.Route.Segments {
		// 完全一致チェック
		// (実務では「エリア指定」などの緩和条件が入ることもありますが、基本はID一致)
		if seg.OriginLocationID == s.OriginID &&
			seg.DestLocationID == s.DestinationID &&
			seg.Mode == s.Mode {
			return true
		}
	}
	return false
}
