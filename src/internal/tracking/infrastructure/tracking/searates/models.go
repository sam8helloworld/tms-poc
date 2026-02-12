package searates

import "time"

// SeaRatesTrackingResponse: SeaRates APIのレスポンス全体
type SeaRatesTrackingResponse struct {
	ContainerNumber string                   `json:"container_number"`
	Status          string                   `json:"status"`
	Events          []SeaRatesContainerEvent `json:"events"`
	Vessel          string                   `json:"vessel"`
	Voyage          string                   `json:"voyage"`
}

// SeaRatesContainerEvent: SeaRates APIの個別イベント構造体
type SeaRatesContainerEvent struct {
	EventType     string    `json:"event_type"`     // "DEPARTURE", "ARRIVAL", "GATE_IN", "GATE_OUT", "LOADED", "DISCHARGED"
	EventDateTime time.Time `json:"event_datetime"` // イベント発生日時
	Location      string    `json:"location"`       // ロケーション名（例: "Tokyo, JP"）
	LocationCode  string    `json:"location_code"`  // UN/LOCODE（例: "JPTYO"）
	Vessel        string    `json:"vessel"`         // 船名
	Voyage        string    `json:"voyage"`         // 航海番号
	Description   string    `json:"description"`    // イベント詳細説明
}
