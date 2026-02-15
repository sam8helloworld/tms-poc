package cmd

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/sam8helloworld/tms-poc/internal/shared/eventbus"
	eventquery "github.com/sam8helloworld/tms-poc/internal/shared/query"

	docapp "github.com/sam8helloworld/tms-poc/internal/document/application/document"
	docpersistence "github.com/sam8helloworld/tms-poc/internal/document/infrastructure/persistence"
	docextractor "github.com/sam8helloworld/tms-poc/internal/document/infrastructure/extractor"
	docquery "github.com/sam8helloworld/tms-poc/internal/document/query"

	networkquery "github.com/sam8helloworld/tms-poc/internal/network/query"

	rateapp "github.com/sam8helloworld/tms-poc/internal/rate/application/rate"
	ratepersistence "github.com/sam8helloworld/tms-poc/internal/rate/infrastructure/persistence"
	ratequery "github.com/sam8helloworld/tms-poc/internal/rate/query"

	shipmentapp "github.com/sam8helloworld/tms-poc/internal/shipment/application/shipment"
	shipmentpersistence "github.com/sam8helloworld/tms-poc/internal/shipment/infrastructure/persistence"
	shipmentquery "github.com/sam8helloworld/tms-poc/internal/shipment/query"

	bidapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/bid"
	tariffapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/tariff"
	sourcingpersistence "github.com/sam8helloworld/tms-poc/internal/sourcing/infrastructure/persistence"
	sourcingquery "github.com/sam8helloworld/tms-poc/internal/sourcing/query"

	trackingapp "github.com/sam8helloworld/tms-poc/internal/tracking/application/tracking"
	trackingdomain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
	trackingprovider "github.com/sam8helloworld/tms-poc/internal/tracking/infrastructure/provider"
	trackingpersistence "github.com/sam8helloworld/tms-poc/internal/tracking/infrastructure/persistence"
	trackingquery "github.com/sam8helloworld/tms-poc/internal/tracking/query"

	operationquery "github.com/sam8helloworld/tms-poc/internal/operation/query"
)

// Dependencies: 全リポジトリ・UseCase・QueryServiceの依存コンテナ
type Dependencies struct {
	// UseCases - Sourcing
	CreateBidContractUC *bidapp.CreateBidContractUseCase
	DeleteBidContractUC *bidapp.DeleteBidContractUseCase
	UpdateContractPeriodUC *bidapp.UpdateContractPeriodUseCase
	RegisterTariffUC    *tariffapp.RegisterTariffUseCase
	AmendTariffUC       *tariffapp.AmendContractTariffUseCase
	AddTariffVersionUC  *tariffapp.AddTariffVersionUseCase
	RemoveTariffUC      *tariffapp.RemoveTariffFromContractUseCase

	// UseCases - Rate
	ApplyContractToRateUC    *rateapp.ApplyContractToRateUseCase
	UpdateRateEntryTariffUC  *rateapp.UpdateRateEntryTariffUseCase

	// UseCases - Tracking
	RegisterTrackingUC *trackingapp.RegisterShipmentTrackingUseCase
	SyncTrackingUC     *trackingapp.SyncTrackingEventsUseCase

	// UseCases - Document
	UploadDocumentUC  *docapp.UploadDocumentUseCase
	ExtractContentUC  *docapp.ExtractDocumentContentUseCase
	ConfirmDocumentUC *docapp.ConfirmDocumentUseCase

	// QueryServices
	SourcingQuery   *sourcingquery.SourcingQueryService
	NetworkQuery    *networkquery.NetworkQueryService
	RateQuery       *ratequery.RateQueryService
	ShipmentQuery   *shipmentquery.ShipmentQueryService
	TrackingQuery   *trackingquery.TrackingQueryService
	DocumentQuery   *docquery.DocumentQueryService
	OperationQuery  *operationquery.OperationQueryService
	EventQuery      *eventquery.EventQueryService
}

// NewDependencies: 全依存の初期化
func NewDependencies(pool *pgxpool.Pool) *Dependencies {
	// Repositories
	contractRepo := sourcingpersistence.NewPostgresContractRepo(pool)
	tariffRepo := sourcingpersistence.NewPostgresTariffRepo(pool)
	rateRepo := ratepersistence.NewPostgresRateRepo(pool)
	shipmentRepo := shipmentpersistence.NewPostgresShipmentRepo(pool)
	trackingRepo := trackingpersistence.NewPostgresTrackingUnitRepo(pool)
	documentRepo := docpersistence.NewPostgresDocumentRepo(pool)

	// EventBus
	bus := eventbus.NewInProcessEventBus()

	// Event Handlers
	trackingRegisteredHandler := shipmentapp.NewTrackingRegisteredHandler(shipmentRepo)
	bus.Subscribe("TrackingRegistered", func(ctx context.Context, event shared.DomainEvent) error {
		if e, ok := event.(trackingdomain.TrackingRegistered); ok {
			return trackingRegisteredHandler.Handle(ctx, e)
		}
		return nil
	})

	// Stubs
	providerRegistry := trackingprovider.NewStubProviderRegistry()
	contentExtractor := docextractor.NewStubDocumentContentExtractor()
	parserFactory := &stubParserFactory{}
	validator := &stubValidator{}

	return &Dependencies{
		// Sourcing UseCases
		CreateBidContractUC:  bidapp.NewCreateBidContractUseCase(contractRepo),
		DeleteBidContractUC:  bidapp.NewDeleteBidContractUseCase(contractRepo),
		UpdateContractPeriodUC: bidapp.NewUpdateContractPeriodUseCase(contractRepo, tariffRepo),
		RegisterTariffUC:     tariffapp.NewRegisterTariffUseCase(parserFactory, validator, tariffRepo, contractRepo),
		AmendTariffUC:        tariffapp.NewAmendContractTariffUseCase(contractRepo, tariffRepo, parserFactory),
		AddTariffVersionUC:   tariffapp.NewAddTariffVersionUseCase(contractRepo, tariffRepo, parserFactory),
		RemoveTariffUC:       tariffapp.NewRemoveTariffFromContractUseCase(contractRepo, tariffRepo),

		// Rate UseCases
		ApplyContractToRateUC:   rateapp.NewApplyContractToRateUseCase(rateRepo, contractRepo, tariffRepo),
		UpdateRateEntryTariffUC: rateapp.NewUpdateRateEntryTariffUseCase(rateRepo, contractRepo, tariffRepo),

		// Tracking UseCases
		RegisterTrackingUC: trackingapp.NewRegisterShipmentTrackingUseCase(trackingRepo, bus),
		SyncTrackingUC:     trackingapp.NewSyncTrackingEventsUseCase(trackingRepo, providerRegistry),

		// Document UseCases
		UploadDocumentUC:  docapp.NewUploadDocumentUseCase(documentRepo),
		ExtractContentUC:  docapp.NewExtractDocumentContentUseCase(documentRepo, contentExtractor),
		ConfirmDocumentUC: docapp.NewConfirmDocumentUseCase(documentRepo),

		// QueryServices
		SourcingQuery:  sourcingquery.NewSourcingQueryService(pool),
		NetworkQuery:   networkquery.NewNetworkQueryService(pool),
		RateQuery:      ratequery.NewRateQueryService(pool),
		ShipmentQuery:  shipmentquery.NewShipmentQueryService(pool),
		TrackingQuery:  trackingquery.NewTrackingQueryService(pool),
		DocumentQuery:  docquery.NewDocumentQueryService(pool),
		OperationQuery: operationquery.NewOperationQueryService(pool),
		EventQuery:     eventquery.NewEventQueryService(pool),
	}
}
