package shipment

// ShipmentStatus: 出荷案件のステータス
type ShipmentStatus string

const (
	StatusPlanned    ShipmentStatus = "PLANNED"     // 計画済み
	StatusBooked     ShipmentStatus = "BOOKED"      // ブッキング済み
	StatusInTransit  ShipmentStatus = "IN_TRANSIT"  // 輸送中
	StatusException  ShipmentStatus = "EXCEPTION"   // 異常・遅延
	StatusCompleted  ShipmentStatus = "COMPLETED"   // 完了
	StatusCancelled  ShipmentStatus = "CANCELLED"   // キャンセル
)

// String: 文字列表現を返す
func (s ShipmentStatus) String() string {
	return string(s)
}

// IsTerminal: 終端状態かどうかを判定
func (s ShipmentStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusCancelled
}
