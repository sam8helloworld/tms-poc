CREATE TYPE contract_status AS ENUM ('DRAFT', 'CONTRACTED', 'EXPIRED', 'CANCELLED');

-- Service Contracts
CREATE TABLE service_contracts (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL,  -- cross-aggregate reference to vendors
    shipper_id  UUID NOT NULL,
    status      contract_status NOT NULL DEFAULT 'DRAFT',
    valid_from  TIMESTAMPTZ NOT NULL,
    valid_to    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contracts_provider_shipper ON service_contracts(provider_id, shipper_id);
CREATE INDEX idx_contracts_status ON service_contracts(status);

-- Tariffs (independent aggregate, references contract by ID)
CREATE TABLE tariffs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_id     UUID NOT NULL,  -- cross-aggregate reference to service_contracts
    name            VARCHAR(255) NOT NULL,
    version         INT NOT NULL DEFAULT 1,
    base_version_id UUID,           -- self-reference for version chain
    effective_from  TIMESTAMPTZ NOT NULL,
    effective_to    TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tariffs_contract ON tariffs(contract_id);
CREATE INDEX idx_tariffs_contract_name ON tariffs(contract_id, name);

-- Tariff Line Items (part of Tariff aggregate)
CREATE TABLE tariff_line_items (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tariff_id   UUID NOT NULL REFERENCES tariffs(id) ON DELETE CASCADE,
    charge_code VARCHAR(100) NOT NULL,
    category    VARCHAR(50) NOT NULL,
    scope       JSONB NOT NULL,  -- polymorphic ServiceScope (LocationService / TransportationService)
    logic       JSONB NOT NULL,  -- polymorphic PricingStrategy
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tariff_line_items_tariff ON tariff_line_items(tariff_id);
