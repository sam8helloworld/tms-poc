package commercial

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
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

// TariffRegistered: 料金表登録イベント
type TariffRegistered struct {
	shared.BaseEvent
	TariffID   uuid.UUID
	TariffName string
	ContractID uuid.UUID
	IsUpdate   bool
}

// NewTariffRegistered: TariffRegisteredイベントを生成
func NewTariffRegistered(contractID, tariffID uuid.UUID, tariffName string, isUpdate bool) TariffRegistered {
	return TariffRegistered{
		BaseEvent:  shared.NewBaseEvent("TariffRegistered", contractID, "ServiceContract"),
		TariffID:   tariffID,
		TariffName: tariffName,
		ContractID: contractID,
		IsUpdate:   isUpdate,
	}
}
