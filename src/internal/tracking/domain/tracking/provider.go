package tracking

import (
	"context"
	"fmt"

	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// TrackingQuery: 外部トラッキングデータ取得のクエリパラメータ
type TrackingQuery struct {
	TrackingNumber string
	Mode           shared.TransportMode
}

// TrackingDataProvider: 外部追跡データ取得のポートインターフェース
// Infrastructure層でSeaRates API等の具象実装を提供する（ACLとして機能）
type TrackingDataProvider interface {
	// SourceType: このプロバイダーが対応するソース種別を返す
	SourceType() TrackingSourceType

	// FetchEvents: 外部システムからトラッキングイベントを取得し、ドメイン型に変換して返す
	// ACL変換はインフラ層の実装側で行う
	FetchEvents(ctx context.Context, query TrackingQuery) ([]TrackingEvent, error)
}

// TrackingDataProviderRegistry: ソース種別によるプロバイダー解決のポートインターフェース
type TrackingDataProviderRegistry interface {
	// GetProvider: ソース種別に対応するプロバイダーを取得
	GetProvider(sourceType TrackingSourceType) (TrackingDataProvider, error)
}

// ErrProviderNotFound: 指定されたソース種別のプロバイダーが見つからない場合のエラー
func ErrProviderNotFound(sourceType TrackingSourceType) error {
	return shared.NewDomainError(
		shared.ErrNotFound,
		fmt.Sprintf("tracking data provider not found for source type: %s", sourceType),
	)
}
