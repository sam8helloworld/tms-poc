CREATE TYPE standard_route_status AS ENUM ('ACTIVE', 'ARCHIVED');

CREATE TABLE standard_routes (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                    VARCHAR(255) NOT NULL,
    shipper_id              UUID NOT NULL,
    origin_location_id      UUID NOT NULL REFERENCES locations(id),
    destination_location_id UUID NOT NULL REFERENCES locations(id),
    status                  standard_route_status NOT NULL DEFAULT 'ACTIVE',
    standard_lead_time_days INT NOT NULL,
    target_cost_amount      NUMERIC(14, 2),
    target_cost_currency    VARCHAR(3),
    valid_from              TIMESTAMPTZ NOT NULL,
    valid_to                TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_standard_routes_shipper ON standard_routes(shipper_id);

CREATE TABLE standard_route_legs (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    standard_route_id    UUID NOT NULL REFERENCES standard_routes(id) ON DELETE CASCADE,
    sequence_order       INT NOT NULL,
    origin_location_id   UUID NOT NULL REFERENCES locations(id),
    dest_location_id     UUID NOT NULL REFERENCES locations(id),
    target_mode          transport_mode NOT NULL,
    standard_transit_days INT NOT NULL,
    master_lane_id       UUID REFERENCES lanes(id),

    CONSTRAINT uq_standard_route_leg_order UNIQUE (standard_route_id, sequence_order)
);

CREATE INDEX idx_standard_route_legs_route ON standard_route_legs(standard_route_id);
