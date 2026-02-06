package shared

import "github.com/shopspring/decimal"

type TransportMode string

const (
	ModeOcean   TransportMode = "OCEAN"
	ModeAir     TransportMode = "AIR"
	ModeTruck   TransportMode = "TRUCK"
	ModeRailway TransportMode = "Railway"
)

// LocationType: 地点の種類 (物理的な施設種別)
type LocationType string

const (
	// 主要な輸送ハブ
	LocTypePort         LocationType = "PORT"          // 港湾 (Sea Port)
	LocTypeAirport      LocationType = "AIRPORT"       // 空港 (Air Port)
	LocTypeRailTerminal LocationType = "RAIL_TERMINAL" // 鉄道ターミナル (Rail Ramp)

	// 内陸・保管施設
	LocTypeWarehouse LocationType = "WAREHOUSE"      // 倉庫 / CFS / DC
	LocTypeYard      LocationType = "CONTAINER_YARD" // コンテナヤード (CY / Depot)

	// 最終地点・その他
	LocTypeDoor   LocationType = "DOOR"   // 工場、店舗、オフィス (特定の住所)
	LocTypeBorder LocationType = "BORDER" // 国境 / 料金所 (Cross-border point)
)

// ValidForMode: その場所が特定の輸送モードで利用可能か判定するヘルパーメソッド
func (lt LocationType) ValidForMode(mode TransportMode) bool {
	switch mode {
	case ModeOcean:
		return lt == LocTypePort || lt == LocTypeYard // 海上は港かCY
	case ModeAir:
		return lt == LocTypeAirport
	case ModeRailway:
		return lt == LocTypeRailTerminal
	case ModeTruck:
		return true // トラックはどこでも行ける
	default:
		return false
	}
}

// TrackingStatus: トラッキングステータス
type TrackingStatus string

const (
	StatusBooked    TrackingStatus = "BOOKED"
	StatusInTransit TrackingStatus = "IN_TRANSIT"
	StatusException TrackingStatus = "EXCEPTION" // 遅延・トラブル
	StatusArrived   TrackingStatus = "ARRIVED"
)

// Decimal: decimal.Decimal のエイリアス
type Decimal = decimal.Decimal

// NewDecimal: decimal.Decimal の生成ヘルパー
func NewDecimal(value int64) Decimal {
	return decimal.NewFromInt(value)
}
