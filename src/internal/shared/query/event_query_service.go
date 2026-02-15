package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sqlcgen"
)

// EventQueryService: ドメインイベントの読み取り専用クエリサービス
type EventQueryService struct {
	q *sqlcgen.Queries
}

func NewEventQueryService(pool *pgxpool.Pool) *EventQueryService {
	return &EventQueryService{q: sqlcgen.New(pool)}
}

func (s *EventQueryService) ListEventsByAggregate(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]sqlcgen.DomainEvent, error) {
	return s.q.ListEventsByAggregate(ctx, sqlcgen.ListEventsByAggregateParams{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
	})
}
