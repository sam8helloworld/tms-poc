package pricing

import (
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/shopspring/decimal"
)

// ShipmentContext: 計算に必要な全てのコンテキスト (物理 + 貨物)
type ShipmentContext struct {
	Route      route.PhysicalRoute // 物理的な移動情報
	Quantity   decimal.Decimal     // 貨物数量
	WeightKG   decimal.Decimal
	VolumeM3   decimal.Decimal
	Attributes map[string]interface{} // 動的属性
}
