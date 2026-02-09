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

## Domain Model

### Go Implementation

The domain model follows Domain-Driven Design (DDD) principles organized by Bounded Contexts:

- **Shared Kernel** (`src/internal/shared/`): Common value objects (Money, DateRange, TransportMode, LocationType)
- **Network BC** (`src/internal/network/domain/route/`): Physical network modeling (Location, Lane, PhysicalRoute, RouteSegment, StandardRoute)
- **Sourcing BC** (`src/internal/sourcing/`): Contracts and pricing
  - `domain/contract/`: ServiceContract, Vendor (LogisticsProvider)
  - `domain/pricing/`: Tariff, TariffLineItem, ServiceScope, PricingStrategy, ShipmentContext
  - `application/bid/`: Bid use cases (CreateBidContract, DeleteContract, UpdateContractPeriod)
  - `application/tariff/`: Tariff use cases (RegisterTariff, AmendTariff, AddTariffVersion, RemoveTariff)
  - `infrastructure/parser/`: TariffParser
- **Rate Management BC** (`src/internal/rate/`): Shipper's internal rates
  - `domain/rate/`: Rate, RateEntry, LogisticsResource
  - `application/rate/`: Rate use cases (ApplyContractToRate, UpdateRateEntryTariff)
- **Shipment BC** (`src/internal/shipment/domain/shipment/`): Shipment execution (Shipment, ShipmentPlan, ShipmentCost)
- **Tracking BC** (`src/internal/tracking/domain/tracking/`): Execution tracking (TrackingUnit, TrackingSegment, ServiceOperator)

Domain model source code is located in `src/internal/`.

### Domain Model Diagram

The current domain model is documented in Mermaid format at `spec/domain-model.md`. The file contains two levels of diagrams:

1. **Context Map** (Abstract): Shows Bounded Contexts, their aggregates, and context mapping patterns (OHS, ACL, Shared Kernel, etc.)
2. **DDD Conceptual Model** (Detailed): Shows all entities, value objects, aggregates, and their relationships

**IMPORTANT**: Whenever you modify the domain model implementation:
1. Update the corresponding Go source files in `src/internal/`
2. Update **BOTH diagrams** in `spec/domain-model.md`:
   - **Context Map**: Update when adding/removing Bounded Contexts, changing aggregates, or modifying relationships between contexts
   - **Detailed Class Diagram**: Update when modifying entities, value objects, attributes, or relationships
3. Ensure both diagrams accurately represent the current architecture and design

The diagrams should be kept in sync with the codebase to serve as living documentation.

#### Domain Model Diagram Style Guide

The domain model diagram follows DDD (Domain-Driven Design) conceptual model principles. When updating the diagram, adhere to these rules:

1. **Focus on Domain Concepts, Not Implementation**:
   - Include only attributes (properties), NOT methods
   - Represent the domain's conceptual model, not the detailed class design
   - Class names should include both Japanese and English: `class Money["金額<br/>(Money)"]`

2. **Aggregate Boundaries**:
   - Use `namespace` blocks to visually group aggregates (e.g., `namespace 商取引集約 { ... }`)
   - Inside aggregates: Use composition (`*--`) to show strong ownership
   - Between aggregates: Use arrows (`-->`) to show ID references (loose coupling)
   - Always include multiplicity on relationships: `"1"`, `"0..n"`, `"1..n"`

3. **Business Rules as Notes**:
   - Use `note for [ClassName]` to document important business rules and constraints
   - Examples: status transitions, validation rules, versioning logic, encapsulation notes
   - Format: `note for ServiceContract "・Rule 1<br/>・Rule 2<br/>・Rule 3"`

4. **What NOT to Include**:
   - UseCases (Application layer)
   - Services (Domain services or Application services)
   - Repositories (Infrastructure interfaces)
   - Adapters (Infrastructure implementations)
   - Method signatures or implementation details

5. **Relationship Types**:
   - `*--` : Composition (aggregate owns entity)
   - `-->` : ID reference (cross-aggregate reference)
   - `..>` : Dependency (uses enumeration or interface)

**Example**:
```mermaid
namespace 商取引集約 {
    class ServiceContract["サービス契約<br/>(ServiceContract)"] {
        <<Aggregate Root>>
        ID: UUID
        ProviderID: UUID
        status: ContractStatus
        tariffs: Tariff[]
    }

    class Tariff["料金表<br/>(Tariff)"] {
        ID: UUID
        Name: String
        Version: Int
    }
}

ServiceContract "1" *-- "0..n" Tariff : 含む
ServiceContract --> LogisticsProvider : 業者参照

note for ServiceContract "・ステータス遷移: DRAFT → CONTRACTED<br/>・status, tariffsフィールドはprivate"
```
