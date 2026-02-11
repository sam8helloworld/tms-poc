package pricing

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// CargoProperties: 貨物の特性属性
type CargoProperties struct {
	HazmatClass     string  // 危険物クラス（空文字 = 非危険物）
	TemperatureMinC *int    // 温度管理（下限）
	TemperatureMaxC *int    // 温度管理（上限）
	IsOversized     bool    // 超過サイズ
	IsFragile       bool    // 易損品
}

// CargoItem: 個別の貨物明細
type CargoItem struct {
	ID          uuid.UUID
	ProductName string
	HSCode      string          // HSコード（関税分類）
	Quantity    decimal.Decimal // 数量
	WeightKG    decimal.Decimal // 重量 (kg)
	VolumeM3    decimal.Decimal // 容積 (m3)
	PackageType string          // パッケージ種別（PALLET, CARTON, DRUM等）
	Properties  CargoProperties
}

// ContainerRequirement: コンテナ要件
type ContainerRequirement struct {
	ContainerType string          // 20DC, 40DC, 40HC等
	Count         decimal.Decimal // 必要コンテナ数
}

// CargoSummary: 貨物集計情報
type CargoSummary struct {
	TotalQuantity         decimal.Decimal
	TotalWeightKG         decimal.Decimal
	TotalVolumeM3         decimal.Decimal
	ChargeableWeightKG    decimal.Decimal // max(実重量, 容積重量)
	ContainerRequirements []ContainerRequirement
}

// defaultVolumetricFactor: 海上輸送のデフォルト容積重量係数 (1 m3 = 1000 kg)
var defaultVolumetricFactor = decimal.NewFromInt(1000)

// NewCargoSummaryFromItems: CargoItemリストから集計情報を生成
func NewCargoSummaryFromItems(items []CargoItem) CargoSummary {
	totalQty := decimal.Zero
	totalWeight := decimal.Zero
	totalVolume := decimal.Zero

	for _, item := range items {
		totalQty = totalQty.Add(item.Quantity)
		totalWeight = totalWeight.Add(item.WeightKG)
		totalVolume = totalVolume.Add(item.VolumeM3)
	}

	// 容積重量 = 容積 × 容積重量係数
	volumetricWeight := totalVolume.Mul(defaultVolumetricFactor)

	// 課金重量 = max(実重量, 容積重量)
	chargeableWeight := totalWeight
	if volumetricWeight.GreaterThan(totalWeight) {
		chargeableWeight = volumetricWeight
	}

	return CargoSummary{
		TotalQuantity:         totalQty,
		TotalWeightKG:         totalWeight,
		TotalVolumeM3:         totalVolume,
		ChargeableWeightKG:    chargeableWeight,
		ContainerRequirements: make([]ContainerRequirement, 0),
	}
}

// Validate: CargoSummaryのバリデーション
func (s CargoSummary) Validate() error {
	if s.TotalQuantity.LessThanOrEqual(decimal.Zero) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "total quantity must be positive")
	}
	if s.TotalWeightKG.LessThan(decimal.Zero) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "total weight must not be negative")
	}
	if s.TotalVolumeM3.LessThan(decimal.Zero) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "total volume must not be negative")
	}
	if s.ChargeableWeightKG.LessThan(decimal.Zero) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "chargeable weight must not be negative")
	}
	return nil
}
