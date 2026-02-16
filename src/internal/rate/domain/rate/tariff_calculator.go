package rate

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// TariffCalculator: Tariff計算のACLインターフェース
// Rate BCのドメイン層で定義し、インフラ層でSourcing BCのTariff.CalculateCharges()に委譲する実装を作る。
// これにより Rate BC は Sourcing BC の内部モデルに依存しない。
type TariffCalculator interface {
	// Calculate: 指定されたTariffのLineItemに対して、ルート・貨物条件で料金を計算する
	Calculate(ctx context.Context, input TariffCalculationInput) (*TariffChargeResult, error)
}

// TariffCalculationInput: Tariff計算の入力（Rate BC固有の型）
type TariffCalculationInput struct {
	TariffID         uuid.UUID           // 対象のTariff ID
	TariffLineItemID uuid.UUID           // 対象のTariffLineItem ID
	Route            route.PhysicalRoute // 物理ルート（区間情報）
	Quantity         decimal.Decimal     // 数量
	WeightKG         decimal.Decimal     // 重量（kg）
	VolumeM3         decimal.Decimal     // 容積（m³）
}

// TariffChargeResult: Tariff計算の結果（Rate BC固有の型）
type TariffChargeResult struct {
	LineItemID uuid.UUID    // 該当するLineItem ID
	ChargeCode string       // 費目コード（OFT, BAFなど）
	Category   string       // カテゴリ（FREIGHT, SURCHARGEなど）
	Amount     shared.Money // 計算済み金額
	Skipped    bool         // スコープ不一致でスキップされたか
	SkipReason string       // スキップされた理由（Skipped=trueの場合）
}
