package bid

import (
	"time"

	"github.com/google/uuid"
)

// CreateBidContractInput: 入札契約作成ユースケースの入力DTO
// 入札プロセスにおいて、各物流業者との契約を作成する
type CreateBidContractInput struct {
	// 入札情報
	BidRequestID uuid.UUID // 入札要求ID（複数の業者への入札をグループ化）

	// 契約当事者情報
	ProviderID uuid.UUID // 物流企業ID（入札参加業者）
	ShipperID  uuid.UUID // 荷主企業ID

	// 契約有効期間
	ValidFrom time.Time
	ValidTo   time.Time

	// メタデータ（オプション）
	BidRequestName string    // 入札要求名（例: "2026年度 北米航路FCL入札"）
	RequestedBy    uuid.UUID // 入札を開始したユーザーのID
	RequestedAt    time.Time // 入札開始日時
	DueDate        time.Time // 入札締切日（業者からの回答期限）

	// 入札対象のルート情報（オプション: 記録用）
	TargetRoutes []BidRouteInfo // 入札対象のルート
}

// BidRouteInfo: 入札対象のルート情報
type BidRouteInfo struct {
	OriginID      uuid.UUID // 出発地ID
	DestinationID uuid.UUID // 到着地ID
	TransportMode string    // "OCEAN", "AIR", "TRUCK"
}

// CreateBidContractOutput: 入札契約作成ユースケースの出力DTO
type CreateBidContractOutput struct {
	// 作成された契約情報
	ContractID      uuid.UUID
	ProviderID      uuid.UUID
	ShipperID       uuid.UUID
	Status          string // "DRAFT"
	ValidFrom       time.Time
	ValidTo         time.Time
	CreatedAt       time.Time
	BidRequestID    uuid.UUID
	BidRequestName  string
	TargetRouteInfo []BidRouteInfo

	// 次のアクションへのガイド
	NextSteps []string // "料金表のアップロードを待つ", "Tariffを登録する", etc.
}
