# ドメインモデル図

## DDD Layered Architecture

このドメインモデルは、国際物流SCMプラットフォームのコア設計を表現しています。
商流（Shipment）と実行（TrackingUnit）を分離し、各時点での費用算出とGap分析を可能にします。

```mermaid
classDiagram
    %% ============================================
    %% Shared (共通値オブジェクト層)
    %% ============================================
    class Money {
        +decimal.Decimal Amount
        +string Currency
        +NewMoney(amount, currency) Money, error
        +ZeroMoney(currency) Money
        +Add(Money) Money, error
        +Sub(Money) Money, error
        +Multiply(decimal.Decimal) Money
        +IsZero() bool
        +IsPositive() bool
        +IsNegative() bool
        +GreaterThan(Money) bool
        +Equals(Money) bool
    }

    class DateRange {
        +time.Time From
        +time.Time To
    }

    class TransportMode {
        <<enumeration>>
        OCEAN
        AIR
        TRUCK
        Railway
    }

    class LocationType {
        <<enumeration>>
        PORT
        AIRPORT
        RAIL_TERMINAL
        WAREHOUSE
        CONTAINER_YARD
        DOOR
        BORDER
        +ValidForMode(TransportMode) bool
    }

    class DomainErrorCode {
        <<enumeration>>
        INVALID_ARGUMENT
        NOT_FOUND
        INVALID_STATE
        BUSINESS_RULE_VIOLATION
        CURRENCY_MISMATCH
    }

    class DomainError {
        +DomainErrorCode Code
        +string Message
        +map Details
        +error Cause
        +Error() string
        +Unwrap() error
        +NewDomainError(code, message) DomainError
        +WithDetail(key, value) DomainError
        +WithCause(error) DomainError
    }

    class DomainEvent {
        <<interface>>
        +EventID() uuid.UUID
        +EventType() string
        +OccurredAt() time.Time
        +AggregateID() uuid.UUID
        +AggregateType() string
    }

    class BaseEvent {
        +uuid.UUID ID
        +string Type
        +time.Time Occurred
        +uuid.UUID AggID
        +string AggType
        +NewBaseEvent(eventType, aggregateID, aggregateType) BaseEvent
    }

    class EventRecorder {
        -DomainEvent[] events
        +RecordEvent(DomainEvent)
        +PullEvents() DomainEvent[]
        +HasEvents() bool
    }

    %% ============================================
    %% Route (ルーティング集約)
    %% ============================================
    class Location {
        +uuid.UUID ID
        +string Name
        +string* UnLocode
        +string CountryCode
        +string Type
    }

    class Lane {
        +uuid.UUID ID
        +uuid.UUID OriginID
        +uuid.UUID DestinationID
        +TransportMode Mode
        +decimal.Decimal DistanceKm
    }

    class PhysicalRoute {
        +uuid.UUID ID
        +uuid.UUID OriginID
        +uuid.UUID DestinationID
        +RouteSegment[] Segments
    }

    class RouteSegment {
        +int SequenceOrder
        +uuid.UUID OriginLocationID
        +LocationType OriginType
        +uuid.UUID DestLocationID
        +LocationType DestType
        +TransportMode Mode
        +decimal.Decimal DistanceKm
        +uuid.UUID* MasterLaneID
    }

    %% ============================================
    %% Commercial (商取引集約)
    %% ============================================
    class ProviderType {
        <<enumeration>>
        CARRIER
        AIRLINE
        TRUCKING_COMPANY
        FORWARDER
        NVOCC
        WAREHOUSE
        CUSTOMS_BROKER
        +IsAssetBased() bool
    }

    class LogisticsProvider {
        +uuid.UUID ID
        +string Name
        +ProviderType Type
    }

    class ContractStatus {
        <<enumeration>>
        DRAFT
        CONTRACTED
        EXPIRED
        CANCELLED
    }

    class ServiceContract {
        <<Aggregate Root>>
        +EventRecorder
        +uuid.UUID ID
        +uuid.UUID ProviderID
        +uuid.UUID ShipperID
        -ContractStatus status
        +DateRange ValidPeriod
        -Tariff[] tariffs
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +NewServiceContract(providerID, shipperID, from, to) ServiceContract, error
        +Status() ContractStatus
        +Tariffs() Tariff[]
        +TariffCount() int
        +AddOrUpdateTariff(Tariff) error
        +RemoveTariff(uuid.UUID) error
        +MarkAsContracted() error
        +MarkAsExpired() error
        +MarkAsCancelled() error
        +Validate() error
        +IsActive() bool
        +IsDraft() bool
    }

    class Tariff {
        +uuid.UUID ID
        +string Name
        +DateRange EffectiveDate
        +TariffLineItem[] LineItems
        +NewTariff(name, from, to) Tariff, error
        +AddLineItem(TariffLineItem) error
        +Validate() error
        +IsEffectiveAt(time.Time) bool
    }

    class TariffLineItem {
        +uuid.UUID ID
        +string ChargeCode
        +string Category
        +ServiceScope Scope
        +PricingStrategy Logic
    }

    class ContractStatusChanged {
        <<Domain Event>>
        +BaseEvent
        +ContractStatus OldStatus
        +ContractStatus NewStatus
    }

    class TariffRegistered {
        <<Domain Event>>
        +BaseEvent
        +uuid.UUID TariffID
        +string TariffName
        +uuid.UUID ContractID
        +bool IsUpdate
    }

    %% ============================================
    %% Rate (社内レート集約)
    %% ============================================
    class RateStatus {
        <<enumeration>>
        DRAFT
        ACTIVE
        EXPIRED
    }

    class Rate {
        <<Aggregate Root>>
        +EventRecorder
        +uuid.UUID ID
        +uuid.UUID ShipperID
        +string Name
        +DateRange ValidPeriod
        -RateStatus status
        -RateEntry[] entries
        +time.Time CreatedAt
        +time.Time UpdatedAt
        +NewRate(shipperID, name, from, to) Rate, error
        +Status() RateStatus
        +Entries() RateEntry[]
        +AddEntry(RateEntry) error
        +RemoveEntry(uuid.UUID) error
        +Activate() error
        +MarkAsExpired() error
        +FindEntriesForRoute(originID, destID, mode) RateEntry[]
    }

    class RateEntry {
        +uuid.UUID ID
        +uuid.UUID ProviderID
        +uuid.UUID ContractID
        +uuid.UUID TariffID
        +RouteScope RouteScope
    }

    class RouteScope {
        <<Value Object>>
        +LocationID* OriginID
        +LocationID* DestinationID
        +TransportMode* TransportMode
        +Matches(originID, destID, mode) bool
    }

    class RateActivated {
        <<Domain Event>>
        +BaseEvent
    }

    class RateEntryAdded {
        <<Domain Event>>
        +BaseEvent
        +uuid.UUID EntryID
    }

    %% ============================================
    %% Shipment (出荷案件集約)
    %% ============================================
    class Shipment {
        <<Aggregate Root>>
        +EventRecorder
        +uuid.UUID ID
        +string ShipmentNo
        +uuid.UUID ShipperID
        +uuid.UUID ConsigneeID
        +ShipmentPlan Plan
        -uuid.UUID[] trackingUnitIDs
        -ShipmentCost cost
        -ShipmentStatus status
        +NewShipment(shipmentNo, shipperID, consigneeID, plan) Shipment, error
        +Status() ShipmentStatus
        +Cost() ShipmentCost
        +TrackingUnitIDs() uuid.UUID[]
        +AddTrackingUnitID(uuid.UUID)
        +UpdateShipmentStatus(ShipmentStatus)
        +SetEstimatedCost(EstimatedCost)
        +SetEstimatedActualCost(EstimatedActualCost)
        +SetActualCost(ActualCost)
    }

    class ShipmentPlan {
        <<Entity>>
        +PhysicalRoute PlannedRoute
        +ShipmentItem[] Items
        +uuid.UUID RateID
        +map TransportRequirements
        +GetTotalWeight() decimal.Decimal
        +GetTotalVolume() decimal.Decimal
        +GetTotalQuantity() decimal.Decimal
    }

    class ShipmentItem {
        <<Entity>>
        +uuid.UUID ID
        +string Commodity
        +string HSCode
        +decimal.Decimal Quantity
        +decimal.Decimal WeightKG
        +decimal.Decimal VolumeM3
        +uuid.UUID* LoadedOnTrackingID
    }

    class ShipmentCost {
        <<Entity>>
        +EstimatedCost* EstimatedCost
        +EstimatedActualCost* EstimatedActualCost
        +ActualCost* ActualCost
        +bool IsFinalized
    }

    class ShipmentStatus {
        <<enumeration>>
        PLANNED
        BOOKED
        IN_TRANSIT
        EXCEPTION
        COMPLETED
        CANCELLED
    }

    class ShipmentCreated {
        <<Domain Event>>
        +BaseEvent
    }

    class ShipmentStatusChanged {
        <<Domain Event>>
        +BaseEvent
        +ShipmentStatus OldStatus
        +ShipmentStatus NewStatus
    }

    %% ============================================
    %% Route Deviation (ルート逸脱分析)
    %% ============================================
    class RouteDeviationAnalysis {
        <<Value Object>>
        +uuid.UUID ShipmentID
        +bool HasDeviation
        +string DeviationReason
        +SegmentMapping[] SegmentMappings
        +RouteSegmentID[] MissingSegments
        +uuid.UUID[] ExtraSegments
    }

    class SegmentMapping {
        <<Value Object>>
        +RouteSegmentID PlannedSegmentID
        +uuid.UUID* ActualSegmentID
        +bool IsMatched
        +DeviationType DeviationType
    }

    class DeviationType {
        <<enumeration>>
        MATCHED
        LOCATION_CHANGED
        SKIPPED
        ADDED
    }

    %% ============================================
    %% Tracking (追跡集約)
    %% ============================================
    class TrackingUnit {
        <<Aggregate Root>>
        +EventRecorder
        +uuid.UUID ID
        +TrackingNumber TrackingNumber
        +uuid.UUID CarrierID
        -TrackingSegment[] segments
        -TrackingStatus currentStatus
        +NewTrackingUnit(trackingNumber, carrierID, segments) TrackingUnit, error
        +CurrentStatus() TrackingStatus
        +Segments() TrackingSegment[]
        +UpdateSegmentStatus(uuid.UUID, TrackingEvent) error
    }

    class TrackingSegment {
        <<Entity>>
        +uuid.UUID ID
        +uuid.UUID ActualOriginLocationID
        +uuid.UUID ActualDestLocationID
        +TransportMode Mode
        +string CarrierTrackingNumber
        +TrackingSourceType PrimarySource
        +TrackingStatus Status
        +TrackingEvent[] Events
        +time.Time* ActualDeparture
        +time.Time* ActualArrival
        +time.Time* EstimatedArrival
    }

    class TrackingEvent {
        <<Value Object>>
        +uuid.UUID ID
        +time.Time Timestamp
        +TrackingSourceType Source
        +string Code
        +string Description
        +string LocationRaw
        +string RawPayload
    }

    class TrackingSourceType {
        <<enumeration>>
        SEARATES_API
        MANUAL_INPUT
        PARTNER_EDI
        DRIVER_APP
        IOT_DEVICE
    }

    class TrackingEventReceived {
        <<Domain Event>>
        +BaseEvent
        +uuid.UUID SegmentID
        +string EventCode
    }

    %% ============================================
    %% CalcParam (計算パラメータ層)
    %% ============================================
    class ShipmentContext {
        +PhysicalRoute Route
        +decimal.Decimal Quantity
        +decimal.Decimal WeightKG
        +decimal.Decimal VolumeM3
        +map Attributes
    }

    %% ============================================
    %% Logic (ビジネスロジック層)
    %% ============================================
    class ServiceScope {
        <<interface>>
        +IsApplicable(ShipmentContext) bool
    }

    class LocationService {
        +uuid.UUID LocationID
        +string ServiceType
        +IsApplicable(ShipmentContext) bool
    }

    class TransportationService {
        +uuid.UUID OriginID
        +uuid.UUID DestinationID
        +TransportMode Mode
        +IsApplicable(ShipmentContext) bool
    }

    class PricingStrategy {
        <<interface>>
        +Calculate(ShipmentContext) Money, error
        +Type() string
    }

    class FlatStrategy {
        +Money Amount
        +Type() string
        +Calculate(ShipmentContext) Money, error
    }

    class CelExpressionStrategy {
        +string Formula
        +string Currency
        +Type() string
        +Calculate(ShipmentContext) Money, error
    }

    class CompositeStrategy {
        +PricingStrategy[] Steps
        +Type() string
        +Calculate(ShipmentContext) Money, error
    }

    %% ============================================
    %% Cost (費用関連)
    %% ============================================
    class EstimatedCost {
        +uuid.UUID RateID
        +CostLineItem[] LineItems
        +Money TotalAmount
        +time.Time CalculatedAt
        +string CalculationBase
    }

    class EstimatedActualCost {
        +uuid.UUID ShipmentID
        +uuid.UUID RateID
        +SegmentCost[] SegmentCosts
        +Money TotalAmount
        +time.Time CalculatedAt
        +string CalculationBase
    }

    class ActualCost {
        +uuid.UUID InvoiceID
        +string InvoiceNo
        +uuid.UUID ProviderID
        +CostLineItem[] LineItems
        +Money TotalAmount
        +time.Time InvoiceDate
    }

    class SegmentCost {
        +uuid.UUID SegmentID
        +int SegmentIndex
        +uuid.UUID OriginLocationID
        +uuid.UUID DestLocationID
        +TransportMode Mode
        +CostLineItem[] LineItems
        +Money TotalAmount
        +SegmentCostStatus CalculationStatus
    }

    class CostLineItem {
        +uuid.UUID ID
        +string ChargeCode
        +string ChargeName
        +string Category
        +Money Amount
    }

    class CostGapAnalysis {
        +uuid.UUID ShipmentID
        +Money EstimatedTotal
        +Money ActualTotal
        +Money TotalGap
        +float64 TotalGapPercentage
        +CostItemGap[] ItemGaps
    }

    class CostItemGap {
        +string ChargeCode
        +Money EstimatedAmount
        +Money ActualAmount
        +Money Gap
        +float64 GapPercentage
    }

    %% ============================================
    %% Service (ドメインサービス層)
    %% ============================================
    class FreightEstimator {
        +Estimate(ShipmentContext, Tariff) EstimatedCost, error
    }

    class CostCalculationService {
        +CalculateEstimatedCost(ShipmentPlan, Tariff) EstimatedCost, error
        +CalculateEstimatedActualCost(Shipment, TrackingUnit[], Tariff) EstimatedActualCost, error
        +AnalyzeCostGap(EstimatedActualCost, ActualCost) CostGapAnalysis, error
    }

    class ShipmentStatusUpdater {
        +UpdateStatus(Shipment, TrackingUnit[])
    }

    class RouteDeviationService {
        +AnalyzeDeviation(Shipment, TrackingUnit[]) RouteDeviationAnalysis
    }

    %% ============================================
    %% UseCase (アプリケーション層)
    %% ============================================
    class RegisterTariffUseCase {
        +Execute(RegisterTariffInput) RegisterTariffOutput, error
    }

    class RegisterTariffInput {
        +io.Reader FileReader
        +string FileFormat
        +string FileName
        +uuid.UUID* ContractID
        +uuid.UUID ProviderID
        +uuid.UUID ShipperID
        +time.Time* ContractValidFrom
        +time.Time* ContractValidTo
        +uuid.UUID UploadedBy
    }

    class RegisterTariffOutput {
        +uuid.UUID ContractID
        +string ContractStatus
        +uuid.UUID TariffID
        +string TariffName
        +time.Time EffectiveFrom
        +time.Time EffectiveTo
        +int LineItemCount
        +bool IsNewContract
        +bool IsUpdatedTariff
        +int TotalTariffCount
    }

    class ApplyContractToRateUseCase {
        +Execute(ApplyContractToRateInput) ApplyContractToRateOutput, error
    }

    class ApplyContractToRateInput {
        +uuid.UUID RateID
        +uuid.UUID ContractID
        +uuid.UUID[] TariffIDs
        +RouteScopeInput RouteScope
    }

    class ApplyContractToRateOutput {
        +uuid.UUID RateID
        +string RateStatus
        +uuid.UUID ContractID
        +uuid.UUID ProviderID
        +AddedEntryDetail[] AddedEntries
        +int TotalEntryCount
    }

    %% ============================================
    %% Adapter (インフラ層インターフェース)
    %% ============================================
    class TariffParser {
        <<interface>>
        +Parse(io.Reader) ParsedTariffData, error
        +SupportedFormats() string[]
    }

    class ParsedTariffData {
        +string TariffName
        +time.Time EffectiveFrom
        +time.Time EffectiveTo
        +ParsedLineItem[] LineItems
    }

    class ParsedLineItem {
        +string ChargeCode
        +string ChargeName
        +string Category
        +string ServiceScopeType
        +map ServiceScopeAttrs
        +string PricingType
        +map PricingAttrs
    }

    class TariffParserFactory {
        <<interface>>
        +GetParser(string) TariffParser, error
    }

    %% ============================================
    %% Repository (リポジトリ層インターフェース)
    %% ============================================
    class ServiceContractRepository {
        <<interface>>
        +Save(ServiceContract) error
        +FindByID(uuid.UUID) ServiceContract, error
        +FindByProviderAndShipper(uuid.UUID, uuid.UUID) ServiceContract[], error
        +FindDraftByProviderAndShipper(uuid.UUID, uuid.UUID) ServiceContract[], error
        +FindActiveByProviderAndShipper(uuid.UUID, uuid.UUID, time.Time) ServiceContract[], error
        +Delete(uuid.UUID) error
    }

    class LogisticsProviderRepository {
        <<interface>>
        +FindByID(uuid.UUID) LogisticsProvider, error
        +FindByName(string) LogisticsProvider[], error
        +Save(LogisticsProvider) error
    }

    class RateRepository {
        <<interface>>
        +Save(Rate) error
        +FindByID(uuid.UUID) Rate, error
        +FindActiveByShipper(uuid.UUID) Rate[], error
        +Delete(uuid.UUID) error
    }

    %% ============================================
    %% Relationships (関連)
    %% ============================================

    %% Shared Layer
    DomainError ..> DomainErrorCode : uses
    BaseEvent ..|> DomainEvent : implements
    EventRecorder o-- DomainEvent : records

    %% Route Aggregate
    PhysicalRoute o-- RouteSegment : contains
    RouteSegment --> Location : origin
    RouteSegment --> Location : destination
    RouteSegment --> Lane : references (optional)
    RouteSegment ..> TransportMode : uses
    RouteSegment ..> LocationType : uses
    Lane --> Location : origin
    Lane --> Location : destination
    Lane ..> TransportMode : uses

    %% Commercial Aggregate
    ServiceContract --> LogisticsProvider : provider
    ServiceContract *-- Tariff : contains (aggregate)
    ServiceContract ..> ContractStatus : uses
    ServiceContract ..> DateRange : uses
    ServiceContract *-- EventRecorder : embeds
    ServiceContract ..> ContractStatusChanged : emits
    ServiceContract ..> TariffRegistered : emits
    Tariff o-- TariffLineItem : contains
    Tariff ..> DateRange : uses
    TariffLineItem --> ServiceScope : has
    TariffLineItem --> PricingStrategy : has
    LogisticsProvider ..> ProviderType : uses
    ContractStatusChanged --> BaseEvent : extends
    TariffRegistered --> BaseEvent : extends

    %% Rate Aggregate
    Rate *-- RateEntry : contains (aggregate)
    Rate ..> RateStatus : uses
    Rate ..> DateRange : uses
    Rate *-- EventRecorder : embeds
    Rate ..> RateActivated : emits
    Rate ..> RateEntryAdded : emits
    RateEntry --> RouteScope : has
    RouteScope --> Location : origin (optional)
    RouteScope --> Location : destination (optional)
    RouteScope ..> TransportMode : uses
    RateEntry --> Tariff : references (TariffID)
    RateEntry --> ServiceContract : references (ContractID)
    RateEntry --> LogisticsProvider : references (ProviderID)
    RateActivated --> BaseEvent : extends
    RateEntryAdded --> BaseEvent : extends

    %% Shipment Aggregate
    ShipmentPlan --> Rate : references (RateID)
    Shipment *-- ShipmentPlan : contains
    Shipment *-- ShipmentCost : contains
    Shipment --> TrackingUnit : references (TrackingUnitIDs)
    Shipment ..> ShipmentStatus : uses
    Shipment *-- EventRecorder : embeds
    Shipment ..> ShipmentCreated : emits
    Shipment ..> ShipmentStatusChanged : emits
    ShipmentPlan *-- ShipmentItem : contains
    ShipmentPlan --> PhysicalRoute : has
    ShipmentItem ..> TrackingUnit : references (LoadedOn)
    ShipmentCost --> EstimatedCost : has
    ShipmentCost --> EstimatedActualCost : has
    ShipmentCost --> ActualCost : has
    ShipmentCreated --> BaseEvent : extends
    ShipmentStatusChanged --> BaseEvent : extends

    %% Tracking Aggregate
    TrackingUnit *-- TrackingSegment : contains
    TrackingUnit ..> TrackingStatus : uses
    TrackingUnit *-- EventRecorder : embeds
    TrackingUnit ..> TrackingEventReceived : emits
    TrackingSegment *-- TrackingEvent : contains
    TrackingSegment --> Location : actualOrigin
    TrackingSegment --> Location : actualDestination
    TrackingSegment ..> TransportMode : uses
    TrackingSegment ..> TrackingSourceType : uses
    TrackingSegment ..> TrackingStatus : uses
    TrackingEvent ..> TrackingSourceType : uses
    TrackingEventReceived --> BaseEvent : extends

    %% Cost Relations
    EstimatedActualCost o-- SegmentCost : contains
    SegmentCost o-- CostLineItem : contains
    EstimatedCost o-- CostLineItem : contains
    ActualCost o-- CostLineItem : contains
    CostGapAnalysis o-- CostItemGap : contains
    CostGapAnalysis --> EstimatedActualCost : analyzes
    CostGapAnalysis --> ActualCost : analyzes

    %% Logic Layer
    LocationService ..|> ServiceScope : implements
    TransportationService ..|> ServiceScope : implements
    FlatStrategy ..|> PricingStrategy : implements
    CelExpressionStrategy ..|> PricingStrategy : implements
    CompositeStrategy ..|> PricingStrategy : implements
    CompositeStrategy o-- PricingStrategy : contains

    %% CalcParam Layer
    ShipmentContext --> PhysicalRoute : has

    %% Service Layer
    FreightEstimator ..> ShipmentContext : uses
    FreightEstimator ..> Tariff : uses
    FreightEstimator ..> EstimatedCost : produces
    CostCalculationService ..> ShipmentPlan : uses
    CostCalculationService ..> Shipment : uses
    CostCalculationService ..> TrackingUnit : uses
    CostCalculationService ..> Tariff : uses
    CostCalculationService ..> EstimatedCost : produces
    CostCalculationService ..> EstimatedActualCost : produces
    CostCalculationService ..> CostGapAnalysis : produces
    ShipmentStatusUpdater ..> Shipment : updates
    ShipmentStatusUpdater ..> TrackingUnit : uses
    RouteDeviationService ..> Shipment : uses
    RouteDeviationService ..> TrackingUnit : uses
    RouteDeviationService ..> RouteDeviationAnalysis : produces

    %% Route Deviation Analysis
    RouteDeviationAnalysis o-- SegmentMapping : contains
    RouteDeviationAnalysis ..> DeviationType : uses
    SegmentMapping --> RouteSegment : references planned
    SegmentMapping --> TrackingSegment : references actual
    SegmentMapping ..> DeviationType : uses

    %% Shared Layer
    FlatStrategy ..> Money : uses
    CelExpressionStrategy ..> Money : uses
    CompositeStrategy ..> Money : uses
    CostLineItem ..> Money : uses
    SegmentCost ..> Money : uses
    EstimatedCost ..> Money : uses
    EstimatedActualCost ..> Money : uses
    ActualCost ..> Money : uses

    %% UseCase Layer
    RegisterTariffUseCase ..> RegisterTariffInput : uses
    RegisterTariffUseCase ..> RegisterTariffOutput : produces
    RegisterTariffUseCase ..> TariffParserFactory : uses
    RegisterTariffUseCase ..> ServiceContractRepository : uses
    RegisterTariffUseCase ..> Tariff : creates
    ApplyContractToRateUseCase ..> ApplyContractToRateInput : uses
    ApplyContractToRateUseCase ..> ApplyContractToRateOutput : produces
    ApplyContractToRateUseCase ..> ServiceContractRepository : uses
    ApplyContractToRateUseCase ..> RateRepository : uses
    ApplyContractToRateUseCase ..> Rate : updates
    ApplyContractToRateUseCase ..> ServiceContract : reads

    %% Adapter Layer
    TariffParser ..> ParsedTariffData : produces
    ParsedTariffData o-- ParsedLineItem : contains
    TariffParserFactory ..> TariffParser : provides

    %% Repository Layer
    ServiceContractRepository ..> ServiceContract : manages
    LogisticsProviderRepository ..> LogisticsProvider : manages
    RateRepository ..> Rate : manages
```

## レイヤー説明

### 1. Shared Layer (共通値オブジェクト層)
- **Money**: 金額と通貨を表現。算術メソッド（Add, Sub, Multiply）と比較メソッド（IsZero, IsPositive, GreaterThan等）を提供
- **DateRange**: 期間を表現（契約有効期限、料金適用期間など）
- **TransportMode**: 輸送モード（海上、航空、トラック、鉄道）
- **LocationType**: 拠点種別（港、空港、倉庫など）
- **TrackingStatus**: トラッキングステータス（BOOKED, IN_TRANSIT, ARRIVED, EXCEPTION）
- **DomainError**: 構造化されたドメインエラー（コード、メッセージ、詳細、原因）
- **DomainEvent / BaseEvent**: ドメインイベントインターフェースと基底実装
- **EventRecorder**: 集約ルートに埋め込んでイベントを記録・取得する仕組み

### 2. Route Aggregate (ルーティング集約)
- **Location**: 物理的な拠点（港、倉庫、空港など）
- **Lane**: 2点間の物理的な輸送路（マスターデータ）
- **PhysicalRoute**: 順序を持った区間の集合体（Origin→Destination）
- **RouteSegment**: A地点からB地点への移動を表す最小単位

### 3. Commercial Aggregate (商取引集約)
- **ServiceContract**: 契約（集約ルート）
  - 入札プロセスにおいて物流企業から提示された料金情報を管理
  - ContractStatus: DRAFT（入札段階）→ CONTRACTED（契約成立）→ EXPIRED/CANCELLED
  - `status`, `tariffs` フィールドはprivateでgetter経由でアクセス
  - EventRecorderを埋め込み、ContractStatusChanged / TariffRegistered イベントを発行
- **LogisticsProvider**: 物流企業（キャリア、フォワーダーなど）
- **Tariff**: 料金表（契約に紐づく料金項目の集合）
  - ServiceContract集約内のエンティティ（ContractIDフィールドは持たない。集約ルートが管理）
- **TariffLineItem**: 個別の料金定義（THC、運賃など）

### 4. Rate Aggregate (社内レート集約)
- **Rate**: 社内レート（集約ルート）
  - 荷主が複数業者のTariffからルート単位で選択・組み合わせた通期レート
  - RateStatus: DRAFT（作成中）→ ACTIVE（使用可能）→ EXPIRED（期限切れ）
  - `status`, `entries` フィールドはprivateでgetter経由でアクセス
  - EventRecorderを埋め込み、RateActivated / RateEntryAdded イベントを発行
- **RateEntry**: レートの構成要素（特定の業者の特定のTariffをまるごと採用）
- **RouteScope**: レートエントリの適用ルート範囲（値オブジェクト）

### 5. Shipment Aggregate (出荷案件集約)
- **Shipment**: 出荷案件（集約ルート）
  - 荷主視点での「1つの仕事」を表現
  - `status`, `cost`, `trackingUnitIDs` フィールドはprivateでgetter経由でアクセス
  - `NewShipment()` ファクトリ関数でバリデーション付き生成
  - EventRecorderを埋め込み、ShipmentCreated / ShipmentStatusChanged イベントを発行
  - ルート逸脱分析はRouteDeviationServiceに委譲
- **ShipmentPlan**: 計画情報（エンティティ）。RateIDで社内レートを参照
- **ShipmentItem**: 貨物明細（エンティティ）
- **ShipmentCost**: 費用情報（エンティティ）

### 6. Tracking Aggregate (追跡集約)
- **TrackingUnit**: 追跡単位（集約ルート）
  - `currentStatus`, `segments` フィールドはprivateでgetter経由でアクセス
  - `NewTrackingUnit()` ファクトリ関数でバリデーション付き生成
  - EventRecorderを埋め込み、TrackingEventReceived イベントを発行
  - 計画への参照を持たない：純粋な実績記録にフォーカス
- **TrackingSegment**: 実際に発生した移動区間（エンティティ）
- **TrackingEvent**: 追跡イベント（値オブジェクト）

### 7. CalcParam Layer (計算パラメータ層)
- **ShipmentContext**: 計算用DTO（計算インターフェース）
  - パッケージ: `calcparam`（Go stdlib `context` との名前衝突を回避）
  - 物理ルート情報、貨物情報（数量、重量、容積）、カスタム属性

### 8. Cost Layer (費用層)
- **EstimatedCost**: 見積費用（計画時点での推定費用）
- **EstimatedActualCost**: 想定実費用（トラッキング実績ベース）
- **ActualCost**: 実請求額（外部インボイスデータ）
- **SegmentCost**: セグメント単位の費用内訳
- **CostLineItem**: 費用明細行
- **CostGapAnalysis**: 費用差異分析結果
- **CostItemGap**: 項目別費用差異

### 9. Logic Layer (ビジネスロジック層)
- **ServiceScope**: 料金適用範囲を判定するインターフェース
  - LocationService: 場所ベースのサービス（THC、保管など）
  - TransportationService: 輸送ベースのサービス（海上運賃、ドレージなど）
- **PricingStrategy**: 料金計算ロジックのインターフェース
  - FlatStrategy: 定額料金（Money.Multiply使用）
  - CelExpressionStrategy: CEL式による動的計算
  - CompositeStrategy: 複数戦略の合成（Money.Add使用、エラーハンドリング付き）

### 10. Service Layer (ドメインサービス層)
- **FreightEstimator**: 見積計算サービス
  - ShipmentContextとTariffから見積費用を計算
- **CostCalculationService**: 費用計算サービス
  - 計画ベースの見積費用算出
  - トラッキング実績ベースの想定費用算出（セグメント単位）
  - 費用差異分析（Money算術メソッド使用）
- **ShipmentStatusUpdater**: ステータス更新サービス
  - TrackingUnitの状態からShipmentのステータスを計算・更新
  - serviceパッケージに配置（集約境界を越えたステータス同期）
- **RouteDeviationService**: ルート逸脱分析サービス
  - Shipmentの計画とTrackingUnitの実績を突合
  - Shipment集約から分離されたドメインサービス

### 11. Route Deviation Analysis (ルート逸脱分析)
- **RouteDeviationAnalysis**: ルート逸脱分析結果（値オブジェクト）
- **SegmentMapping**: 計画セグメントと実績セグメントの対応関係
- **RouteDeviationService.AnalyzeDeviation()**: ドメインサービスメソッド

### 12. UseCase Layer (アプリケーション層)
- **RegisterTariffUseCase**: 料金表登録ユースケース
  - contract.Status() / contract.TariffCount() / contract.Tariffs() getter使用
- **ApplyContractToRateUseCase**: 契約反映ユースケース
  - CONTRACTED状態の契約から料金表（一部または全部）をDRAFT状態のRateに反映
  - contract.IsActive() でCONTRACTED状態を検証、contract.Tariffs() で料金表を取得

### 13. Adapter Layer (インフラ層インターフェース)
- **TariffParser**: 料金表ファイル解析インターフェース
- **ParsedTariffData**: 解析された料金表データ（中間データ構造）
- **TariffParserFactory**: パーサーファクトリー

### 14. Repository Layer (リポジトリ層インターフェース)
- **ServiceContractRepository**: ServiceContract集約のリポジトリ
- **LogisticsProviderRepository**: LogisticsProvider集約のリポジトリ
- **RateRepository**: Rate集約のリポジトリ

## 主要な設計パターン

1. **Aggregate分離と境界**:
   - ShipmentとTrackingUnitを独立した集約として分離
   - ServiceContractを集約ルートとしてTariffを管理
   - Rateを集約ルートとしてRateEntryを管理（複数業者のTariffを組み合わせた社内レート）
   - 集約間の参照はIDのみ（疎結合）
2. **カプセル化**:
   - 集約ルートの重要フィールドはprivate（小文字）
   - getter メソッドで安全にアクセス（コレクションはコピー返却）
   - ファクトリ関数でバリデーション付き生成
3. **ドメインイベント**:
   - EventRecorderを集約ルートに埋め込み
   - 状態変更時にイベントを発行（ContractStatusChanged, ShipmentCreated等）
   - PullEvents()で後続処理に引き渡し
4. **構造化エラー**:
   - DomainError型でコード、メッセージ、詳細、原因を保持
   - errors.New()ではなくNewDomainError()を使用
   - IsCode()ヘルパーでエラー種別判定
5. **関心の分離**: 計画と実績の明確な分離
   - **Shipment**: 計画（PlannedRoute）の管理者
   - **TrackingUnit**: 実績の記録者（計画への参照を持たない）
   - **RouteDeviationService**: 計画と実績の突合（ドメインサービス）
   - **ShipmentStatusUpdater**: 集約を越えたステータス同期（ドメインサービス）
6. **Strategy Pattern**: PricingStrategyによる計算ロジックの分離
7. **Composite Pattern**: CompositeStrategyによる料金の合成
8. **Value Object**: Money（算術メソッド付き）, DateRange（不変性、等価性）
9. **Domain Service**: 複数集約にまたがるロジック
   - FreightEstimator, CostCalculationService, ShipmentStatusUpdater, RouteDeviationService
10. **Factory Pattern**:
    - NewServiceContract(), NewShipment(), NewTrackingUnit(): バリデーション付き生成
    - NewTariff(): 不変条件を保証したTariff生成
    - TariffParserFactory: ファイル形式に応じたパーサー選択
11. **UseCase Pattern (Application Service)**:
    - RegisterTariffUseCase: getter経由でのprivateフィールドアクセス
12. **Adapter Pattern (Hexagonal Architecture)**: TariffParser
13. **Repository Pattern**: ドメイン層ではインターフェースのみ定義
14. **State Pattern**: ContractStatus, ShipmentStatus のライフサイクル管理

## 費用計算の流れ

### 1. 計画時点（見積）
```
ShipmentPlan → CostCalculationService → EstimatedCost
                ↓ uses
              Tariff
```

### 2. トラッキング時点（想定実費用）
```
Shipment + TrackingUnit[] → CostCalculationService → EstimatedActualCost
                              ↓ uses                    ↓ contains
                            Tariff                   SegmentCost[]
```

### 3. 請求時点（Gap分析）
```
EstimatedActualCost + ActualCost → CostCalculationService → CostGapAnalysis
                                                               ↓ contains
                                                            CostItemGap[]
```

## セグメント単位の費用計算ステータス

- **COMPLETED**: 完了済み（実績ベースで確定）
- **IN_PROGRESS**: 進行中（按分計算による推定）
- **PLANNED**: 未着手（計画ベースの推定）
- **NOT_APPLICABLE**: 適用対象外
