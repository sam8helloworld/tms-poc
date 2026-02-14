package rate

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ReconstructRate: 永続化層からRateを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructRate(
	id, shipperID uuid.UUID,
	name string,
	status RateStatus,
	validPeriod shared.DateRange,
	entries []*RateEntry,
	createdAt, updatedAt time.Time,
) *Rate {
	return &Rate{
		ID:          id,
		ShipperID:   shipperID,
		Name:        name,
		status:      status,
		ValidPeriod: validPeriod,
		entries:     entries,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
