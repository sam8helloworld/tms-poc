package pricing

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

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
		BaseEvent:  shared.NewBaseEvent("TariffRegistered", tariffID, "Tariff"),
		TariffID:   tariffID,
		TariffName: tariffName,
		ContractID: contractID,
		IsUpdate:   isUpdate,
	}
}

// TariffAmended: CONTRACTED状態の契約に対する料金表改定イベント
type TariffAmended struct {
	shared.BaseEvent
	TariffID     uuid.UUID
	TariffName   string
	NewVersion   int
	BaseTariffID uuid.UUID
	ContractID   uuid.UUID
}

// NewTariffAmended: TariffAmendedイベントを生成
func NewTariffAmended(contractID, tariffID uuid.UUID, tariffName string, newVersion int, baseTariffID uuid.UUID) TariffAmended {
	return TariffAmended{
		BaseEvent:    shared.NewBaseEvent("TariffAmended", tariffID, "Tariff"),
		TariffID:     tariffID,
		TariffName:   tariffName,
		NewVersion:   newVersion,
		BaseTariffID: baseTariffID,
		ContractID:   contractID,
	}
}

// TariffRemoved: 料金表削除イベント
type TariffRemoved struct {
	shared.BaseEvent
	TariffID   uuid.UUID
	TariffName string
	ContractID uuid.UUID
}

// NewTariffRemoved: TariffRemovedイベントを生成
func NewTariffRemoved(contractID, tariffID uuid.UUID, tariffName string) TariffRemoved {
	return TariffRemoved{
		BaseEvent:  shared.NewBaseEvent("TariffRemoved", tariffID, "Tariff"),
		TariffID:   tariffID,
		TariffName: tariffName,
		ContractID: contractID,
	}
}
