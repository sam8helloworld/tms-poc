package route

import "github.com/google/uuid"

type LocationID uuid.UUID

// Location: 物理的な拠点 (Port, Warehouse)
type Location struct {
	ID          LocationID
	Name        string
	UnLocode    *string
	CountryCode string
	Type        string
}
