-- ENUM types for Network BC
CREATE TYPE location_type AS ENUM (
    'PORT', 'AIRPORT', 'RAIL_TERMINAL', 'WAREHOUSE',
    'CONTAINER_YARD', 'DOOR', 'BORDER'
);

CREATE TYPE transport_mode AS ENUM (
    'OCEAN', 'AIR', 'TRUCK', 'Railway'
);

-- Locations (Nodes in logistics network)
CREATE TABLE locations (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    un_locode   VARCHAR(5),
    country_code VARCHAR(2) NOT NULL,
    type        location_type NOT NULL,
    geom        GEOMETRY(POINT, 4326),
    attributes  JSONB DEFAULT '{}'::JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_locations_un_locode ON locations(un_locode);
CREATE INDEX idx_locations_country ON locations(country_code);
CREATE INDEX idx_locations_geom ON locations USING GIST(geom);

-- Lanes (Edges in logistics network)
CREATE TABLE lanes (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    origin_id      UUID NOT NULL REFERENCES locations(id),
    destination_id UUID NOT NULL REFERENCES locations(id),
    mode           transport_mode NOT NULL,
    distance_km    NUMERIC(12, 2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_lanes UNIQUE (origin_id, destination_id, mode)
);

CREATE INDEX idx_lanes_origin ON lanes(origin_id);
CREATE INDEX idx_lanes_destination ON lanes(destination_id);
