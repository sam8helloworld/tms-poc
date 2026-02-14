package route

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ReconstructStandardRoute: 永続化層からStandardRouteを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructStandardRoute(
	id StandardRouteID,
	name string,
	shipperID uuid.UUID,
	originLocationID, destinationLocationID LocationID,
	legs []StandardRouteLeg,
	status StandardRouteStatus,
	standardLeadTimeDays int,
	targetCost *shared.Money,
	validPeriod shared.DateRange,
	createdAt, updatedAt time.Time,
) *StandardRoute {
	return &StandardRoute{
		ID:                    id,
		Name:                  name,
		ShipperID:             shipperID,
		OriginLocationID:      originLocationID,
		DestinationLocationID: destinationLocationID,
		legs:                  legs,
		status:                status,
		StandardLeadTimeDays:  standardLeadTimeDays,
		TargetCost:            targetCost,
		ValidPeriod:           validPeriod,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}
}
