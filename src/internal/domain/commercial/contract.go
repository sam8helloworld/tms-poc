package commercial

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ServiceContract: 契約
type ServiceContract struct {
	ID          uuid.UUID
	ProviderID  uuid.UUID
	ShipperID   uuid.UUID
	ValidPeriod shared.DateRange
	Tariffs     []*Tariff
}
