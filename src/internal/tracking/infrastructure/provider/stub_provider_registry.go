package provider

import (
	"context"
	"fmt"

	"github.com/sam8helloworld/tms-poc/internal/shared"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// StubProviderRegistry: 開発・テスト用のTrackingDataProviderRegistryスタブ
type StubProviderRegistry struct{}

// NewStubProviderRegistry: StubProviderRegistryのコンストラクタ
func NewStubProviderRegistry() *StubProviderRegistry {
	return &StubProviderRegistry{}
}

// GetProvider: スタブプロバイダーを返す
func (r *StubProviderRegistry) GetProvider(sourceType domain.TrackingSourceType) (domain.TrackingDataProvider, error) {
	return &stubProvider{sourceType: sourceType}, nil
}

// stubProvider: 空のイベントを返すスタブプロバイダー
type stubProvider struct {
	sourceType domain.TrackingSourceType
}

func (p *stubProvider) SourceType() domain.TrackingSourceType {
	return p.sourceType
}

func (p *stubProvider) FetchEvents(ctx context.Context, query domain.TrackingQuery) ([]domain.TrackingEvent, error) {
	_ = ctx
	_ = query
	fmt.Printf("[STUB] FetchEvents called for source=%s, tracking=%s (returning empty)\n", p.sourceType, query.TrackingNumber)
	return nil, nil
}

// Verify interface compliance
var _ domain.TrackingDataProviderRegistry = (*StubProviderRegistry)(nil)
var _ domain.TrackingDataProvider = (*stubProvider)(nil)

// NoopEventPublisher: 何もしないDomainEventPublisher
type NoopEventPublisher struct{}

func NewNoopEventPublisher() *NoopEventPublisher {
	return &NoopEventPublisher{}
}

func (p *NoopEventPublisher) Publish(ctx context.Context, events []shared.DomainEvent) error {
	return nil
}

var _ shared.DomainEventPublisher = (*NoopEventPublisher)(nil)
