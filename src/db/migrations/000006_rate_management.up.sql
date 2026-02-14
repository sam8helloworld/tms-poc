CREATE TYPE rate_status AS ENUM ('DRAFT', 'ACTIVE', 'EXPIRED');

-- Rates (shipper's internal rate aggregation)
CREATE TABLE rates (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipper_id  UUID NOT NULL,
    name        VARCHAR(255) NOT NULL,
    status      rate_status NOT NULL DEFAULT 'DRAFT',
    valid_from  TIMESTAMPTZ NOT NULL,
    valid_to    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rates_shipper ON rates(shipper_id);
CREATE INDEX idx_rates_status ON rates(status);

-- Rate Entries (part of Rate aggregate)
CREATE TABLE rate_entries (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rate_id       UUID NOT NULL REFERENCES rates(id) ON DELETE CASCADE,
    provider_id   UUID NOT NULL,    -- cross-aggregate reference to vendors
    contract_id   UUID NOT NULL,    -- cross-aggregate reference to service_contracts
    tariff_id     UUID NOT NULL,    -- cross-aggregate reference to tariffs
    origin_id     UUID,             -- nullable: nil = all origins
    destination_id UUID,            -- nullable: nil = all destinations
    transport_mode transport_mode   -- nullable: nil = all modes
);

CREATE INDEX idx_rate_entries_rate ON rate_entries(rate_id);
CREATE INDEX idx_rate_entries_route ON rate_entries(origin_id, destination_id, transport_mode);

-- Logistics Resources (rate/planning context view of provider)
CREATE TABLE logistics_resources (
    provider_id  UUID PRIMARY KEY,  -- cross-aggregate reference to vendors
    name         VARCHAR(255) NOT NULL,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    capabilities JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
