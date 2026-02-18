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

	// OperatorVendorID: この料金項目の実際の作業業者（任意）
	// nilの場合は契約者(ServiceContract.ProviderID)と同一、または未指定
	// 例: 国際輸送=船社ID, 倉庫作業=倉庫業者ID
	OperatorVendorID *uuid.UUID
}
