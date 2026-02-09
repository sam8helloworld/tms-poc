package contract

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ContractStatusChanged: 契約ステータス変更イベント
type ContractStatusChanged struct {
	shared.BaseEvent
	OldStatus ContractStatus
	NewStatus ContractStatus
}

// NewContractStatusChanged: ContractStatusChangedイベントを生成
func NewContractStatusChanged(contractID uuid.UUID, oldStatus, newStatus ContractStatus) ContractStatusChanged {
	return ContractStatusChanged{
		BaseEvent: shared.NewBaseEvent("ContractStatusChanged", contractID, "ServiceContract"),
		OldStatus: oldStatus,
		NewStatus: newStatus,
	}
}
