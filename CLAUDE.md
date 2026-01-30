# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a POC for an international logistics SCM platform that manages physical logistics networks (graph structure) and commercial flows (rates, contracts, costs) using PostgreSQL with PostGIS.

## Database Architecture

### Core Data Model

The database follows a two-layer architecture:

1. **Physical Network Layer** - Manages logistics infrastructure:
   - `locations`: Nodes representing ports, airports, warehouses, regions, countries. Uses hierarchical structure via `parent_location_id` (e.g., "Oi Wharf" → "Tokyo Port" → "Kanto Region")
   - `transport_edges`: Edges connecting locations, representing physical/logical routes for routing algorithms

2. **Commercial & Rates Layer** - Manages commercial transactions:
   - `partners`: Trading partners (carriers, forwarders, trucking companies)
   - `rate_cards`: Contract definitions (partner, route, validity period, conditions)
   - `rate_items`: Detailed pricing with support for complex calculation logic via JSONB

3. **Normalization Layer** - Standardizes charge codes:
   - `charge_codes`: Standard internal charge codes
   - `charge_code_mappings`: Maps external charge names to internal codes

### Key Design Patterns

- **JSONB for flexibility**:
  - `locations.attributes`: Port-specific metadata (draft limits, facilities)
  - `rate_cards.conditions`: Business rules (no hazmat, space-dependent)
  - `rate_items.tier_matrix`: Complex tariff structures (distance-based pricing)

- **Hierarchical location lookup**: When no rate exists for specific location, fall back to parent/grandparent locations

- **PostGIS spatial features**: Uses `GEOMETRY(POINT)` and `GEOMETRY(LINESTRING)` for geospatial queries

- **DATERANGE for validity**: Efficiently query rates valid for specific periods using `&&` operator

## Common Commands

### Database Management

Start database:
```bash
docker compose up -d
```

Stop database:
```bash
docker compose down
```

Stop and remove volumes (re-applies init scripts):
```bash
docker compose down -v
```

Check database status:
```bash
docker compose ps
docker compose logs postgres
```

### Connecting to Database

Via psql in container:
```bash
docker compose exec postgres psql -U postgres -d tms_db
```

Via psql from host:
```bash
psql -h localhost -p 5432 -U postgres -d tms_db
```

Default credentials: `postgres/postgres` for database `tms_db`

### Running SQL Files

Using helper script (recommended):
```bash
./run-sql.sh path/to/file.sql
```

Direct execution:
```bash
docker compose exec -T postgres psql -U postgres -d tms_db < path/to/file.sql
```

### Initialization Scripts

SQL files in `init/` directory are executed automatically on first container startup (when data volume is empty). Execution order:
1. `001_create_graph_tables.sql` - Creates extensions, ENUM types, locations, transport_edges
2. `002_create_rate_table.sql` - Creates partners, rate_cards, rate_items
3. `003_create_charge_mapping_table.sql` - Creates charge_codes, charge_code_mappings

To re-apply init scripts, use `docker compose down -v && docker compose up -d`

## Database Schema Details

### ENUM Types

- `location_type`: PORT, AIRPORT, RAIL_YARD, WAREHOUSE, FACTORY, REGION, COUNTRY, CITY
- `transport_mode`: OCEAN_FCL, OCEAN_LCL, AIR, TRUCK, RAIL, BARGE
- `currency_code`: USD, JPY, EUR, CNY
- `charge_unit`: PER_CONTAINER, PER_SHIPMENT, PER_KG, PER_M3
- `container_type`: 20DC, 40DC, 40HC, LCL, BULK
- `charge_category`: FREIGHT_BASIC, SURCHARGE_FUEL, SURCHARGE_CCY, ORIGIN_LOCAL, DEST_LOCAL, DUTY_TAX

### Important Indexes

- `locations`: UN/LOCODE lookup, parent hierarchy, spatial (GIST) for geom
- `transport_edges`: Source/target lookups, composite for routing
- `rate_cards`: Origin/dest/partner composite, validity (GIST) for date ranges
- `charge_code_mappings`: Partner/input_text for fast lookup during data import

## Development Notes

When working with rate calculations, `tier_matrix` JSONB structure for distance-based pricing:
```json
{
  "logic": "step",
  "steps": [
    {"max_km": 10, "price": 18060},
    {"max_km": 20, "price": 20160}
  ],
  "over_max_rule": {"unit_km": 20, "add_price": 4140}
}
```

The `calculation_type` field in `rate_items` can be: FIXED, DISTANCE_TIER, WEIGHT_TIER, PERCENTAGE
