package contract

type ProviderType string

const (
	// Carrier (Asset-based): 船・飛行機・トラックを自社で保有する実運送人
	ProviderTypeCarrier ProviderType = "CARRIER"          // 船会社 (Maersk, ONE)
	ProviderTypeAirline ProviderType = "AIRLINE"          // 航空会社 (JAL, ANA)
	ProviderTypeTrucker ProviderType = "TRUCKING_COMPANY" // トラック会社

	// Intermediary (Non-Asset based): 荷主とキャリアの間に入る業者
	ProviderTypeForwarder ProviderType = "FORWARDER" // フォワーダー (KWE, Nippon Express)
	ProviderTypeNVOCC     ProviderType = "NVOCC"     // NVOCC (Forwarderと兼ねる場合も多いが区分する場合)

	// Specific Services
	ProviderTypeWarehouse ProviderType = "WAREHOUSE"      // 倉庫業者
	ProviderTypeBroker    ProviderType = "CUSTOMS_BROKER" // 通関業者
)

// IsAssetBased: 自社資産(船・車)を持っているか判定するドメインロジック
func (pt ProviderType) IsAssetBased() bool {
	switch pt {
	case ProviderTypeCarrier, ProviderTypeAirline, ProviderTypeTrucker:
		return true
	default:
		return false
	}
}
