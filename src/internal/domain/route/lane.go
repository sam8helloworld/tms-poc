package route

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
	"github.com/shopspring/decimal"
)

// Lane: 2点間の物理的な輸送路
type Lane struct {
	ID            uuid.UUID
	OriginID      uuid.UUID
	DestinationID uuid.UUID
	Mode          shared.TransportMode
	DistanceKm    decimal.Decimal
}
