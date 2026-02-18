package pricing

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// RateableLineItem: Rate BC向けのBC中立なDTO
// Tariff LineItemから抽出したルート・単価情報
type RateableLineItem struct {
	LineItemID       uuid.UUID
	TariffID         uuid.UUID
	ChargeCode       string
	Category         string
	OriginID         *uuid.UUID
	DestinationID    *uuid.UUID
	TransportMode    *string
	UnitPrice        *shared.Money // FlatStrategy の場合のみ。他は nil
	OperatorVendorID *uuid.UUID    // 実際の作業業者（任意）
}

// ExtractRateableItems: TariffのLineItemsからRate BCが消費可能な形式に変換
// TransportationService scope + FlatStrategy の組み合わせでルート・単価を抽出
func ExtractRateableItems(tariff *Tariff) []RateableLineItem {
	items := make([]RateableLineItem, 0, len(tariff.LineItems))
	for _, li := range tariff.LineItems {
		item := RateableLineItem{
			LineItemID:       li.ID,
			TariffID:         tariff.ID,
			ChargeCode:       li.ChargeCode,
			Category:         li.Category,
			OperatorVendorID: li.OperatorVendorID,
		}

		// Scope からルート情報を抽出
		switch s := li.Scope.(type) {
		case TransportationService:
			oid := uuid.UUID(s.OriginID)
			did := uuid.UUID(s.DestinationID)
			mode := string(s.Mode)
			item.OriginID = &oid
			item.DestinationID = &did
			item.TransportMode = &mode
		case *TransportationService:
			oid := uuid.UUID(s.OriginID)
			did := uuid.UUID(s.DestinationID)
			mode := string(s.Mode)
			item.OriginID = &oid
			item.DestinationID = &did
			item.TransportMode = &mode
		}

		// Logic から単価を抽出（FlatStrategy のみ）
		switch s := li.Logic.(type) {
		case *FlatStrategy:
			price := s.Amount
			item.UnitPrice = &price
		}

		items = append(items, item)
	}
	return items
}
