package rate

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// LogisticsResource: レート・計画コンテキストにおける物流企業（能力とコストの提供者）
// "誰が運べるか"ではなく"何を運べるか"に焦点を当てたモデル
// RateEntryから参照される
type LogisticsResource struct {
	ProviderID uuid.UUID // 契約コンテキストのVendor.IDと紐づく

	// 計画コンテキストの関心事：能力（Capability）情報
	Name         string
	Capabilities []ResourceCapability // 提供可能な輸送能力

	// レート参照の利便性
	IsAvailable bool // 現在利用可能か（契約状態により変動）
}

// ResourceCapability: 物流リソースが提供する能力定義
type ResourceCapability struct {
	RouteScope    RouteScope           // どのルート範囲に対応できるか
	TransportMode shared.TransportMode // 輸送モード
	Capacity      CapacitySpec         // 能力スペック

	// コスト特性
	RateLevel      RateLevel // 料金レベル（HIGH/MEDIUM/LOW）
	LeadTimeDays   int       // 標準リードタイム
	ReliabilityPct int       // 定時性 (%)
}

// CapacitySpec: 能力スペック
type CapacitySpec struct {
	MaxWeightKG  *float64 // 最大積載重量 (nil=無制限)
	MaxVolumeM3  *float64 // 最大容積 (nil=無制限)
	ContainerTypes []string // 対応コンテナタイプ (20DC, 40HC etc.)
	SpecialHandling []string // 特殊対応 ("REEFER", "HAZMAT", "OVERSIZED")
}

// RateLevel: 料金レベル（計画時の簡易判断用）
type RateLevel string

const (
	RateLevelHigh   RateLevel = "HIGH"   // 高価格帯（速達、高品質）
	RateLevelMedium RateLevel = "MEDIUM" // 中価格帯
	RateLevelLow    RateLevel = "LOW"    // 低価格帯（エコノミー）
)

// NewLogisticsResource: LogisticsResourceのファクトリ関数
func NewLogisticsResource(providerID uuid.UUID, name string) *LogisticsResource {
	return &LogisticsResource{
		ProviderID:   providerID,
		Name:         name,
		Capabilities: make([]ResourceCapability, 0),
		IsAvailable:  true,
	}
}

// AddCapability: 能力を追加
func (lr *LogisticsResource) AddCapability(capability ResourceCapability) {
	lr.Capabilities = append(lr.Capabilities, capability)
}

// CanHandleRoute: 指定されたルートを扱えるか判定
func (lr *LogisticsResource) CanHandleRoute(
	originID route.LocationID,
	destID route.LocationID,
	mode shared.TransportMode,
	weightKG *float64,
	volumeM3 *float64,
) bool {
	if !lr.IsAvailable {
		return false
	}

	for _, cap := range lr.Capabilities {
		// ルート範囲チェック
		if !cap.RouteScope.Matches(originID, destID, mode) {
			continue
		}

		// 輸送モードチェック
		if cap.TransportMode != mode {
			continue
		}

		// 重量チェック
		if weightKG != nil && cap.Capacity.MaxWeightKG != nil {
			if *weightKG > *cap.Capacity.MaxWeightKG {
				continue
			}
		}

		// 容積チェック
		if volumeM3 != nil && cap.Capacity.MaxVolumeM3 != nil {
			if *volumeM3 > *cap.Capacity.MaxVolumeM3 {
				continue
			}
		}

		return true
	}

	return false
}

// GetCapabilityForRoute: ルートに合致する能力を取得
func (lr *LogisticsResource) GetCapabilityForRoute(
	originID route.LocationID,
	destID route.LocationID,
	mode shared.TransportMode,
) *ResourceCapability {
	for _, cap := range lr.Capabilities {
		if cap.RouteScope.Matches(originID, destID, mode) && cap.TransportMode == mode {
			return &cap
		}
	}
	return nil
}

// MarkUnavailable: 利用不可にする（契約期限切れ等）
func (lr *LogisticsResource) MarkUnavailable() {
	lr.IsAvailable = false
}

// MarkAvailable: 利用可能にする
func (lr *LogisticsResource) MarkAvailable() {
	lr.IsAvailable = true
}
