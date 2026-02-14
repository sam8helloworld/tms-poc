CREATE TYPE provider_type AS ENUM (
    'CARRIER', 'FORWARDER', 'WAREHOUSE', 'CUSTOMS_BROKER', 'TRUCKER'
);

CREATE TABLE vendors (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             VARCHAR(255) NOT NULL,
    type             provider_type NOT NULL,
    credit_rating    VARCHAR(10) NOT NULL DEFAULT 'BBB',
    payment_days     INT NOT NULL DEFAULT 30,
    payment_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    preferred_vendor BOOLEAN NOT NULL DEFAULT FALSE,
    capabilities     JSONB NOT NULL DEFAULT '[]'::JSONB,
    contacts         JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vendors_name ON vendors(name);
CREATE INDEX idx_vendors_type ON vendors(type);
