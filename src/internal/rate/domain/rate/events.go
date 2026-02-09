package rate

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// RateActivated: レート有効化イベント（DRAFT → ACTIVE）
type RateActivated struct {
	shared.BaseEvent
}

// NewRateActivated: RateActivatedイベントを生成
func NewRateActivated(rateID uuid.UUID) RateActivated {
	return RateActivated{
		BaseEvent: shared.NewBaseEvent("RateActivated", rateID, "Rate"),
	}
}

// RateEntryAdded: レートエントリ追加イベント
type RateEntryAdded struct {
	shared.BaseEvent
	EntryID uuid.UUID
}

// NewRateEntryAdded: RateEntryAddedイベントを生成
func NewRateEntryAdded(rateID, entryID uuid.UUID) RateEntryAdded {
	return RateEntryAdded{
		BaseEvent: shared.NewBaseEvent("RateEntryAdded", rateID, "Rate"),
		EntryID:   entryID,
	}
}

// RateEntryTariffReplaced: レートエントリのTariff差し替えイベント
type RateEntryTariffReplaced struct {
	shared.BaseEvent
	EntryID     uuid.UUID
	OldTariffID uuid.UUID
	NewTariffID uuid.UUID
}

// NewRateEntryTariffReplaced: RateEntryTariffReplacedイベントを生成
func NewRateEntryTariffReplaced(rateID, entryID, oldTariffID, newTariffID uuid.UUID) RateEntryTariffReplaced {
	return RateEntryTariffReplaced{
		BaseEvent:   shared.NewBaseEvent("RateEntryTariffReplaced", rateID, "Rate"),
		EntryID:     entryID,
		OldTariffID: oldTariffID,
		NewTariffID: newTariffID,
	}
}
