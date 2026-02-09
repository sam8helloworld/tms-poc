package contract

import "github.com/google/uuid"

// LogisticsProvider: 物流企業
type LogisticsProvider struct {
	ID   uuid.UUID
	Name string
	Type ProviderType // CARRIER, FORWARDER
}
