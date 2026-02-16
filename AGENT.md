# AGENT.md

This file provides guidance to AI coding assistants when working with code in this repository.

## Project Overview

This is a POC for an international logistics SCM platform that manages physical logistics networks (graph structure) and commercial flows (rates, contracts, costs) using PostgreSQL with PostGIS.

## Tech Stack

- **Language**: Go 1.24
- **Database**: PostgreSQL 16 + PostGIS 3.4
- **DB Driver**: pgx/v5 (pgxpool)
- **Migration**: golang-migrate (Up/Down SQL)
- **Query**: sqlc
- **Container**: Docker Compose

## Common Commands

All Makefile targets are in `src/Makefile` (run from `src/` directory):

```bash
# Build
cd src && go build ./...

# Docker
cd src && make docker-up      # Start PostgreSQL
cd src && make docker-down    # Stop PostgreSQL

# Migration
cd src && make migrate-up     # Apply all migrations
cd src && make migrate-down   # Rollback all migrations
cd src && make migrate-create # Create new migration (prompts for name)

# Code Generation
cd src && make sqlc-generate  # Generate Go code from SQL queries
```

### Connecting to Database

```bash
docker compose exec postgres psql -U postgres -d tms_db
# or from host:
psql -h localhost -p 5432 -U postgres -d tms_db
```

Default credentials: `postgres/postgres` for database `tms_db`

## Domain Model

### Go Implementation

The domain model follows Domain-Driven Design (DDD) principles organized by Bounded Contexts:

- **Shared Kernel** (`src/internal/shared/`): Common value objects (Money, DateRange, TransportMode, LocationType, DocType, TradeDirection)
- **Network BC** (`src/internal/network/`): Physical network modeling
  - `domain/route/`: Location, Lane, PhysicalRoute, RouteSegment, StandardRoute
  - `infrastructure/persistence/`: PostgresLocationRepo, PostgresLaneRepo, PostgresStandardRouteRepo
- **Sourcing BC** (`src/internal/sourcing/`): Contracts and pricing
  - `domain/contract/`: ServiceContract, Vendor (LogisticsProvider)
  - `domain/pricing/`: Tariff, TariffLineItem, ServiceScope, PricingStrategy, ShipmentContext
  - `application/bid/`: Bid use cases (CreateBidContract, DeleteContract, UpdateContractPeriod)
  - `application/tariff/`: Tariff use cases (RegisterTariff, AmendTariff, AddTariffVersion, RemoveTariff)
  - `infrastructure/parser/`: TariffParser
  - `infrastructure/persistence/`: PostgresVendorRepo, PostgresContractRepo, PostgresTariffRepo
- **Rate Management BC** (`src/internal/rate/`): Shipper's internal rates
  - `domain/rate/`: Rate, RateEntry, LogisticsResource
  - `application/rate/`: Rate use cases (ApplyContractToRate, UpdateRateEntryTariff)
  - `infrastructure/persistence/`: PostgresRateRepo
- **Shipment BC** (`src/internal/shipment/`): Shipment execution
  - `domain/shipment/`: Shipment, ShipmentPlan, ShipmentCost, ShipmentExecution, Milestone
  - `infrastructure/persistence/`: PostgresShipmentRepo
- **Tracking BC** (`src/internal/tracking/`): Execution tracking
  - `domain/tracking/`: TrackingUnit, TrackingSegment, ServiceOperator
  - `infrastructure/persistence/`: PostgresTrackingUnitRepo
- **Operation BC** (`src/internal/operation/`): SOP management
  - `domain/sop/`: SOPDefinition, SOPInstance, SOPTask
  - `infrastructure/persistence/`: PostgresSOPDefinitionRepo, PostgresSOPInstanceRepo
- **Document BC** (`src/internal/document/`): Document management
  - `domain/document/`: Document, DocumentContent, DocumentReview
  - `application/document/`: UploadDocument, ExtractContent, ConfirmDocument
  - `infrastructure/persistence/`: PostgresDocumentRepo

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
3. If the change affects DB schema (new table/column, type change, new relationship, new ENUM value, etc.), create a new migration file via `cd src && make migrate-create` and update:
   - Migration SQL (`src/db/migrations/`)
   - sqlc query SQL (`src/db/queries/`)
   - Repository implementation (`src/internal/{bc}/infrastructure/persistence/`)
   - Reconstruct function if private fields changed (`src/internal/{bc}/domain/{aggregate}/reconstruct.go`)
4. Ensure all diagrams, DB schema, and code accurately represent the current architecture and design

The diagrams, DB schema, and code should be kept in sync to serve as living documentation.

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

## Database Architecture

### Migration Management

Migrations are managed by golang-migrate in `src/db/migrations/`. Each migration has an up and down SQL file.

| # | Migration | Tables |
|---|-----------|--------|
| 000001 | extensions | uuid-ossp, postgis |
| 000002 | network_locations_lanes | locations, lanes |
| 000003 | network_standard_routes | standard_routes, standard_route_legs |
| 000004 | sourcing_vendors | vendors |
| 000005 | sourcing_contracts_tariffs | service_contracts, tariffs, tariff_line_items |
| 000006 | rate_management | rates, rate_entries, logistics_resources |
| 000007 | shipments | shipments + 6 child tables (segments, items, tracking_units, milestones, cost_line_items, cost_summaries) |
| 000008 | tracking | tracking_units, tracking_segments, tracking_events, service_operators |
| 000009 | operation_sop | sop_definitions, sop_step_definitions, sop_instances, sop_tasks |
| 000010 | documents | documents, document_reviews |
| 000011 | domain_events | domain_events (outbox) |

### Key DB Design Patterns

- **Aggregate boundary = transaction boundary**: Each Repository.Save() uses a single transaction
- **Cross-aggregate references**: ID reference only, no foreign keys between aggregates
- **Intra-aggregate CASCADE**: Child tables use `ON DELETE CASCADE` to parent
- **JSONB for polymorphic/denormalized data**: vendor capabilities/contacts, tariff scope/logic, document content, milestone payload, SOP linked_document_ids, service operator contacts/metrics/channels, logistics resource capabilities
- **Normalized child tables for high-volume/queryable data**: tariff_line_items, tracking_events, shipment_milestones, document_reviews, rate_entries
- **Upsert pattern**: `INSERT ON CONFLICT DO UPDATE` for idempotent Save
- **Append-only pattern**: `ON CONFLICT DO NOTHING` for events, milestones, reviews
- **Domain Event Outbox**: `domain_events` table for transactional outbox pattern

### sqlc Queries

Query definitions are in `src/db/queries/`. After modifying queries, run `cd src && make sqlc-generate`.

### Repository Implementation Conventions

- Location: `src/internal/{bc}/infrastructure/persistence/postgres_{aggregate}_repo.go`
- Constructor: `NewPostgres{Aggregate}Repo(pool *pgxpool.Pool)`
- Reconstruct functions: `src/internal/{bc}/domain/{aggregate}/reconstruct.go` — bypasses domain validation for DB reconstruction of aggregates with private fields
- Common infrastructure: `src/pkg/postgres/` (pool.go, tx_manager.go)

## Scenarios

POC シナリオは `src/cmd/tms/cmd/scenarios/` に格納されています。

**シナリオを追加・変更する際は必ず `src/cmd/tms/cmd/scenarios/README.md` を更新すること**:
- 新規シナリオ追加: シナリオ名・概要・業務フロー表・マスターデータ・出力例を追記
- 既存シナリオ変更: 変更内容(ステップ説明、フロー表など)を README に反映
- README は実装と常に同期された状態を維持する
