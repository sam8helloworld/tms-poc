package eventbus

import (
	"context"
	"sync"

	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// EventHandler: ドメインイベントを処理するハンドラ
type EventHandler func(ctx context.Context, event shared.DomainEvent) error

// InProcessEventBus: インプロセスの同期イベントバス
// CLI用途では同期ディスパッチで十分
type InProcessEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// NewInProcessEventBus: InProcessEventBusのコンストラクタ
func NewInProcessEventBus() *InProcessEventBus {
	return &InProcessEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Subscribe: イベントタイプに対するハンドラを登録
func (b *InProcessEventBus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish: ドメインイベントを発行し、登録されたハンドラを同期的に呼び出す
func (b *InProcessEventBus) Publish(ctx context.Context, events []shared.DomainEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, event := range events {
		handlers, ok := b.handlers[event.EventType()]
		if !ok {
			continue
		}
		for _, handler := range handlers {
			if err := handler(ctx, event); err != nil {
				return err
			}
		}
	}
	return nil
}
