package commercial

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/pricing"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/scope"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// Tariff: 料金表 (Rate Book / Catalog)
// 1つの契約の下にぶら下がる、大量の料金項目の集合体
type Tariff struct {
	ID            uuid.UUID
	ContractID    uuid.UUID
	Name          string // e.g. "2026 Japan Export"
	EffectiveDate shared.DateRange

	// LineItems: 個別の料金定義のリスト
	LineItems []TariffLineItem
}

// TariffLineItem: 1行の料金定義 (The Rate)
type TariffLineItem struct {
	ID         uuid.UUID
	ChargeCode string // "OFT", "THC"
	Category   string // FREIGHT, LOCAL

	// Scope: どこに適用されるか (ドメイン用語で定義)
	Scope scope.ServiceScope

	// Logic: いくらか (Strategy Pattern)
	Logic pricing.PricingStrategy
}
