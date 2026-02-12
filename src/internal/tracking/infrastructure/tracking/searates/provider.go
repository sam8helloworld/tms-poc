package searates

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
)

// SeaRatesProvider: SeaRates APIのTrackingDataProvider実装（stub）
// 実際のAPI呼び出しは行わず、空リストを返すスタブ実装
// ACL変換メソッドを備え、将来的に実API統合時にそのまま利用可能
type SeaRatesProvider struct{}

// NewSeaRatesProvider: SeaRatesProviderのコンストラクタ
func NewSeaRatesProvider() *SeaRatesProvider {
	return &SeaRatesProvider{}
}

// SourceType: このプロバイダーが対応するソース種別を返す
func (p *SeaRatesProvider) SourceType() domain.TrackingSourceType {
	return domain.SourceSeaRates
}

// FetchEvents: 外部システムからトラッキングイベントを取得（stub: 空リスト返却）
func (p *SeaRatesProvider) FetchEvents(ctx context.Context, query domain.TrackingQuery) ([]domain.TrackingEvent, error) {
	// TODO: 実際のSeaRates API呼び出しを実装
	// response, err := p.callSeaRatesAPI(ctx, query.TrackingNumber)
	// if err != nil {
	// 	return nil, err
	// }
	// return p.convertToDomainEvents(response), nil

	return []domain.TrackingEvent{}, nil
}

// convertToDomainEvent: SeaRatesイベントをドメインのTrackingEventに変換（ACL）
func convertToDomainEvent(src SeaRatesContainerEvent) domain.TrackingEvent {
	return domain.TrackingEvent{
		ID:          uuid.New(),
		Timestamp:   src.EventDateTime,
		Source:      domain.SourceSeaRates,
		Code:        mapEventCode(src.EventType),
		Description: src.Description,
		LocationRaw: src.Location,
	}
}

// convertToDomainEvents: SeaRatesレスポンスの全イベントをドメイン型に変換
func convertToDomainEvents(response *SeaRatesTrackingResponse) []domain.TrackingEvent {
	events := make([]domain.TrackingEvent, 0, len(response.Events))
	for _, src := range response.Events {
		events = append(events, convertToDomainEvent(src))
	}
	return events
}

// mapEventCode: SeaRatesのイベントコードをドメイン標準コードにマッピング
func mapEventCode(seaRatesEventType string) string {
	mapping := map[string]string{
		"DEPARTURE":  "DEPT",
		"ARRIVAL":    "ARRI",
		"GATE_IN":    "GTIN",
		"GATE_OUT":   "GTOT",
		"LOADED":     "LOAD",
		"DISCHARGED": "DISC",
		"TRANSSHIP":  "TRSH",
	}
	if code, ok := mapping[seaRatesEventType]; ok {
		return code
	}
	return seaRatesEventType
}
