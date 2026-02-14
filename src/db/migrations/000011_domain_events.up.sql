-- Domain Events outbox table
CREATE TABLE domain_events (
    id             UUID PRIMARY KEY,
    event_type     VARCHAR(200) NOT NULL,
    aggregate_id   UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    payload        JSONB NOT NULL,
    occurred_at    TIMESTAMPTZ NOT NULL,
    published      BOOLEAN NOT NULL DEFAULT FALSE,
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_domain_events_aggregate ON domain_events(aggregate_id, aggregate_type);
CREATE INDEX idx_domain_events_unpublished ON domain_events(published) WHERE published = FALSE;
CREATE INDEX idx_domain_events_occurred ON domain_events(occurred_at);
