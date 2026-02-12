package tracking

import (
	"fmt"

	"github.com/sam8helloworld/tms-poc/internal/shared"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// ProviderRegistry: TrackingDataProviderRegistryのインフラ実装
// map[TrackingSourceType]TrackingDataProviderによるシンプルな実装
type ProviderRegistry struct {
	providers map[domain.TrackingSourceType]domain.TrackingDataProvider
}

// NewProviderRegistry: ProviderRegistryのコンストラクタ
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[domain.TrackingSourceType]domain.TrackingDataProvider),
	}
}

// Register: プロバイダーを登録
func (r *ProviderRegistry) Register(provider domain.TrackingDataProvider) {
	r.providers[provider.SourceType()] = provider
}

// GetProvider: ソース種別に対応するプロバイダーを取得
func (r *ProviderRegistry) GetProvider(sourceType domain.TrackingSourceType) (domain.TrackingDataProvider, error) {
	provider, ok := r.providers[sourceType]
	if !ok {
		return nil, shared.NewDomainError(
			shared.ErrNotFound,
			fmt.Sprintf("tracking data provider not found for source type: %s", sourceType),
		)
	}
	return provider, nil
}
