CREATE TYPE shipment_status AS ENUM (
    'PLANNED', 'BOOKED', 'IN_TRANSIT', 'ARRIVED', 'DELIVERED', 'CANCELLED'
);

CREATE TYPE cost_category AS ENUM (
    'ESTIMATED', 'ESTIMATED_ACTUAL', 'ACTUAL'
);

-- Shipments (aggregate root)
CREATE TABLE shipments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_no         VARCHAR(100) NOT NULL UNIQUE,
    shipper_id          UUID NOT NULL,
    consignee_id        UUID NOT NULL,
    status              shipment_status NOT NULL DEFAULT 'PLANNED',
    standard_route_id   UUID,           -- cross-aggregate reference to standard_routes
    rate_id             UUID,           -- cross-aggregate reference to rates
    origin_location_id  UUID NOT NULL,
    dest_location_id    UUID NOT NULL,
    transport_requirements JSONB DEFAULT '{}'::JSONB,
    cost_is_finalized   BOOLEAN NOT NULL DEFAULT FALSE,
    cost_finalized_at   TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shipments_shipper ON shipments(shipper_id);
CREATE INDEX idx_shipments_no ON shipments(shipment_no);
CREATE INDEX idx_shipments_status ON shipments(status);

-- Shipment Planned Route Segments (part of aggregate)
CREATE TABLE shipment_planned_route_segments (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id           UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    sequence_order        INT NOT NULL,
    origin_location_id    UUID NOT NULL,
    origin_type           location_type NOT NULL,
    dest_location_id      UUID NOT NULL,
    dest_type             location_type NOT NULL,
    mode                  transport_mode NOT NULL,
    distance_km           NUMERIC(12, 2),
    master_lane_id        UUID,

    CONSTRAINT uq_shipment_segment_order UNIQUE (shipment_id, sequence_order)
);

CREATE INDEX idx_shipment_segments_shipment ON shipment_planned_route_segments(shipment_id);

-- Shipment Items (part of aggregate)
CREATE TABLE shipment_items (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id             UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    commodity               VARCHAR(255) NOT NULL,
    hs_code                 VARCHAR(20),
    quantity                NUMERIC(14, 4) NOT NULL,
    weight_kg               NUMERIC(14, 4) NOT NULL,
    volume_m3               NUMERIC(14, 4) NOT NULL,
    package_type            VARCHAR(50),
    loaded_on_tracking_id   UUID,
    attributes              JSONB DEFAULT '{}'::JSONB
);

CREATE INDEX idx_shipment_items_shipment ON shipment_items(shipment_id);

-- Shipment Tracking Unit references (part of aggregate)
CREATE TABLE shipment_tracking_units (
    shipment_id      UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    tracking_unit_id UUID NOT NULL,

    PRIMARY KEY (shipment_id, tracking_unit_id)
);

-- Shipment Milestones (part of aggregate, append-only)
CREATE TABLE shipment_milestones (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id        UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    milestone_type     VARCHAR(100) NOT NULL,
    occurred_at        TIMESTAMPTZ NOT NULL,
    recorded_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_document_id UUID NOT NULL,
    source_doc_type    VARCHAR(50) NOT NULL,
    payload            JSONB NOT NULL,
    sequence           INT NOT NULL
);

CREATE INDEX idx_milestones_shipment ON shipment_milestones(shipment_id);
CREATE INDEX idx_milestones_type ON shipment_milestones(shipment_id, milestone_type);

-- Shipment Cost Line Items (part of aggregate)
CREATE TABLE shipment_cost_line_items (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id   UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    cost_category cost_category NOT NULL,
    charge_code   VARCHAR(100) NOT NULL,
    charge_name   VARCHAR(255),
    category      VARCHAR(50),
    amount        NUMERIC(14, 2) NOT NULL,
    currency      VARCHAR(3) NOT NULL,
    quantity      NUMERIC(14, 4),
    unit_price    NUMERIC(14, 2),
    unit_currency VARCHAR(3),
    applied_scope VARCHAR(255),
    remarks       TEXT,
    -- for ESTIMATED_ACTUAL: segment breakdown
    segment_id    UUID,
    segment_index INT
);

CREATE INDEX idx_cost_lines_shipment ON shipment_cost_line_items(shipment_id);
CREATE INDEX idx_cost_lines_category ON shipment_cost_line_items(shipment_id, cost_category);

-- Shipment Cost Summaries (part of aggregate)
CREATE TABLE shipment_cost_summaries (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shipment_id   UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    cost_category cost_category NOT NULL,
    rate_id       UUID,
    invoice_id    UUID,
    invoice_no    VARCHAR(100),
    provider_id   UUID,
    total_amount  NUMERIC(14, 2) NOT NULL,
    total_currency VARCHAR(3) NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    calculation_base VARCHAR(50),

    CONSTRAINT uq_cost_summary UNIQUE (shipment_id, cost_category)
);
