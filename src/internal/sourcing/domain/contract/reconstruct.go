package contract

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ReconstructServiceContract: 永続化層からServiceContractを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructServiceContract(
	id, providerID, shipperID uuid.UUID,
	status ContractStatus,
	validPeriod shared.DateRange,
	createdAt, updatedAt time.Time,
) *ServiceContract {
	return &ServiceContract{
		ID:          id,
		ProviderID:  providerID,
		ShipperID:   shipperID,
		status:      status,
		ValidPeriod: validPeriod,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
