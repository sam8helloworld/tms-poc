CREATE TYPE tracking_status AS ENUM ('BOOKED', 'IN_TRANSIT', 'EXCEPTION', 'ARRIVED');

CREATE TYPE tracking_number_type AS ENUM (
    'CONTAINER', 'AIRWAY_BILL', 'BILL_OF_LADING', 'BOOKING_NUMBER'
);

CREATE TYPE tracking_source_type AS ENUM (
    'SEARATES_API', 'MANUAL_INPUT', 'PARTNER_EDI', 'DRIVER_APP', 'IOT_DEVICE'
);

CREATE TYPE operator_role AS ENUM (
    'TRANSPORTER', 'WAREHOUSE', 'CUSTOMS_BROKER',
    'DELIVERY_AGENT', 'PACKING_SERVICE', 'INSPECTOR'
);

-- Tracking Units (aggregate root)
CREATE TABLE tracking_units (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tracking_number       VARCHAR(100) NOT NULL,
    tracking_number_type  tracking_number_type NOT NULL,
    carrier_id            UUID NOT NULL,
    current_status        tracking_status NOT NULL DEFAULT 'BOOKED',
    last_updated          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tracking_units_number ON tracking_units(tracking_number);
CREATE INDEX idx_tracking_units_carrier ON tracking_units(carrier_id);

-- Tracking Segments (part of aggregate)
CREATE TABLE tracking_segments (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tracking_unit_id        UUID NOT NULL REFERENCES tracking_units(id) ON DELETE CASCADE,
    actual_origin_location_id UUID NOT NULL,
    actual_dest_location_id   UUID NOT NULL,
    mode                    transport_mode NOT NULL,
    carrier_tracking_number VARCHAR(100),
    primary_source          tracking_source_type NOT NULL,
    status                  tracking_status NOT NULL DEFAULT 'BOOKED',
    actual_departure        TIMESTAMPTZ,
    actual_arrival          TIMESTAMPTZ,
    estimated_arrival       TIMESTAMPTZ
);

CREATE INDEX idx_tracking_segments_unit ON tracking_segments(tracking_unit_id);

-- Tracking Events (part of aggregate, append-only)
CREATE TABLE tracking_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    segment_id  UUID NOT NULL REFERENCES tracking_segments(id) ON DELETE CASCADE,
    timestamp   TIMESTAMPTZ NOT NULL,
    source      tracking_source_type NOT NULL,
    code        VARCHAR(50) NOT NULL,
    description TEXT,
    location_raw VARCHAR(255),
    raw_payload TEXT
);

CREATE INDEX idx_tracking_events_segment ON tracking_events(segment_id);
CREATE INDEX idx_tracking_events_time ON tracking_events(segment_id, timestamp);

-- Service Operators (execution context view of provider)
CREATE TABLE service_operators (
    provider_id           UUID PRIMARY KEY,
    name                  VARCHAR(255) NOT NULL,
    role                  operator_role NOT NULL,
    operational_contacts  JSONB NOT NULL DEFAULT '[]'::JSONB,
    performance_metrics   JSONB NOT NULL DEFAULT '{}'::JSONB,
    integration_channels  JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
