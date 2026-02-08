package route

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ==========================================
// Domain Events: StandardRoute
// ==========================================

// StandardRouteCreated: 標準ルートが作成された
type StandardRouteCreated struct {
	shared.BaseEvent
}

func NewStandardRouteCreated(standardRouteID uuid.UUID) StandardRouteCreated {
	return StandardRouteCreated{
		BaseEvent: shared.NewBaseEvent("StandardRouteCreated", standardRouteID, "StandardRoute"),
	}
}

// StandardRouteArchived: 標準ルートがアーカイブされた
type StandardRouteArchived struct {
	shared.BaseEvent
}

func NewStandardRouteArchived(standardRouteID uuid.UUID) StandardRouteArchived {
	return StandardRouteArchived{
		BaseEvent: shared.NewBaseEvent("StandardRouteArchived", standardRouteID, "StandardRoute"),
	}
}
