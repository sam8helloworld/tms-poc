package route

import "github.com/google/uuid"

// Location: 物理的な拠点 (Port, Warehouse)
type Location struct {
	ID          uuid.UUID
	Name        string
	UnLocode    *string
	CountryCode string
	Type        string
}
