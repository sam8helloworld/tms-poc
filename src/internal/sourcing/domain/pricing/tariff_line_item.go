package pricing

import (
	"github.com/google/uuid"
)

// TariffLineItem: 1行の料金定義 (The Rate)
type TariffLineItem struct {
	ID         uuid.UUID
	ChargeCode string // "OFT", "THC"
	Category   string // FREIGHT, LOCAL

	// Scope: どこに適用されるか (ドメイン用語で定義)
	Scope ServiceScope

	// Logic: いくらか (Strategy Pattern)
	Logic PricingStrategy
}
