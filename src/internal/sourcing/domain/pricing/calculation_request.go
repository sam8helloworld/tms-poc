package pricing

import (
	"time"

	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// CalculationConditions: 計算条件
type CalculationConditions struct {
	DesiredShipDate     *time.Time        // 出荷希望日
	Incoterms           string            // FOB, CIF, DDP等
	SpecialRequirements []string          // 特殊要件のリスト
	Attributes          map[string]string // 拡張用属性
}

// CalculationRequest: 料金計算リクエストVO
type CalculationRequest struct {
	Route      route.PhysicalRoute
	Items      []CargoItem
	Summary    CargoSummary
	Conditions CalculationConditions
}

// NewCalculationRequest: CargoItemsから自動集計するファクトリ
func NewCalculationRequest(
	r route.PhysicalRoute,
	items []CargoItem,
	conditions CalculationConditions,
) (*CalculationRequest, error) {
	if len(items) == 0 {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "at least one cargo item is required")
	}

	summary := NewCargoSummaryFromItems(items)
	if err := summary.Validate(); err != nil {
		return nil, err
	}

	return &CalculationRequest{
		Route:      r,
		Items:      items,
		Summary:    summary,
		Conditions: conditions,
	}, nil
}

// NewCalculationRequestWithSummary: Summary直接指定（概算用）
func NewCalculationRequestWithSummary(
	r route.PhysicalRoute,
	summary CargoSummary,
	conditions CalculationConditions,
) (*CalculationRequest, error) {
	if err := summary.Validate(); err != nil {
		return nil, err
	}

	return &CalculationRequest{
		Route:      r,
		Items:      nil,
		Summary:    summary,
		Conditions: conditions,
	}, nil
}

// toShipmentContext: 内部計算エンジン用の表現に変換（package-private）
func (r *CalculationRequest) toShipmentContext() ShipmentContext {
	attrs := make(map[string]interface{})

	// Conditions.Attributesをマージ
	for k, v := range r.Conditions.Attributes {
		attrs[k] = v
	}

	// 標準フィールドを属性にマッピング
	if r.Conditions.Incoterms != "" {
		attrs["incoterms"] = r.Conditions.Incoterms
	}
	if r.Conditions.DesiredShipDate != nil {
		attrs["desiredShipDate"] = r.Conditions.DesiredShipDate.Format(time.RFC3339)
	}
	if len(r.Conditions.SpecialRequirements) > 0 {
		attrs["specialRequirements"] = r.Conditions.SpecialRequirements
	}
	// 課金重量を属性に追加（PricingStrategyで使用可能にする）
	if !r.Summary.ChargeableWeightKG.IsZero() {
		attrs["chargeableWeightKG"] = r.Summary.ChargeableWeightKG.String()
	}

	// Quantity: ContainerRequirementsがあればコンテナ数合計、なければTotalQuantity
	quantity := r.Summary.TotalQuantity
	if len(r.Summary.ContainerRequirements) > 0 {
		total := decimal.Zero
		for _, cr := range r.Summary.ContainerRequirements {
			total = total.Add(cr.Count)
		}
		quantity = total
	}

	return ShipmentContext{
		Route:      r.Route,
		Quantity:   quantity,
		WeightKG:   r.Summary.TotalWeightKG,
		VolumeM3:   r.Summary.TotalVolumeM3,
		Attributes: attrs,
	}
}
