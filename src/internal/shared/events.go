package shared

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DomainEvent: ドメインイベントインターフェース
type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	OccurredAt() time.Time
	AggregateID() uuid.UUID
	AggregateType() string
}

// BaseEvent: ドメインイベントの基底構造体
type BaseEvent struct {
	ID            uuid.UUID
	Type          string
	Occurred      time.Time
	AggID         uuid.UUID
	AggType       string
}

// NewBaseEvent: BaseEventのファクトリ
func NewBaseEvent(eventType string, aggregateID uuid.UUID, aggregateType string) BaseEvent {
	return BaseEvent{
		ID:            uuid.New(),
		Type:          eventType,
		Occurred:      time.Now(),
		AggID:         aggregateID,
		AggType:       aggregateType,
	}
}

func (e BaseEvent) EventID() uuid.UUID      { return e.ID }
func (e BaseEvent) EventType() string        { return e.Type }
func (e BaseEvent) OccurredAt() time.Time    { return e.Occurred }
func (e BaseEvent) AggregateID() uuid.UUID   { return e.AggID }
func (e BaseEvent) AggregateType() string    { return e.AggType }

// EventRecorder: 集約ルートに埋め込んでイベントを記録する
type EventRecorder struct {
	events []DomainEvent
}

// RecordEvent: イベントを記録
func (r *EventRecorder) RecordEvent(event DomainEvent) {
	r.events = append(r.events, event)
}

// PullEvents: 記録されたイベントを取得しクリアする
func (r *EventRecorder) PullEvents() []DomainEvent {
	events := r.events
	r.events = nil
	return events
}

// HasEvents: 未処理のイベントがあるか判定
func (r *EventRecorder) HasEvents() bool {
	return len(r.events) > 0
}

// DomainEventPublisher: ドメインイベントを外部に発行するインターフェース
// コンテキスト間の結果整合性を実現するために使用する
// Infrastructure層で具象実装を提供する（メッセージキュー、インプロセスバス等）
type DomainEventPublisher interface {
	// Publish: ドメインイベントを発行する
	Publish(ctx context.Context, events []DomainEvent) error
}
