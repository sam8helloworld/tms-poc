package scenarios

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	networkquery "github.com/sam8helloworld/tms-poc/internal/network/query"
	rateapp "github.com/sam8helloworld/tms-poc/internal/rate/application/rate"
	ratequery "github.com/sam8helloworld/tms-poc/internal/rate/query"
	bidapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/bid"
	tariffapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/tariff"
	sourcingquery "github.com/sam8helloworld/tms-poc/internal/sourcing/query"
)

// Scenario: シナリオインターフェース
type Scenario interface {
	Name() string
	Description() string
	Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error
}

// ScenarioDeps: シナリオ実行に必要なUseCase・QueryService群
// cmd.Dependencies からシナリオで使うものだけを受け取る
type ScenarioDeps struct {
	// UseCases - Sourcing
	CreateBidContractUC    *bidapp.CreateBidContractUseCase
	AwardBidContractUC     *bidapp.AwardBidContractUseCase
	RegisterTariffDirectUC *tariffapp.RegisterTariffDirectUseCase
	AmendTariffDirectUC    *tariffapp.AmendContractTariffDirectUseCase

	// UseCases - Rate
	CreateRateUC            *rateapp.CreateRateUseCase
	ActivateRateUC          *rateapp.ActivateRateUseCase
	ApplyContractToRateUC   *rateapp.ApplyContractToRateUseCase
	UpdateRateEntryTariffUC *rateapp.UpdateRateEntryTariffUseCase
	SimulateRateCostUC      *rateapp.SimulateRateCostUseCase

	// QueryServices（各ステップ後のデータ確認用）
	NetworkQuery  *networkquery.NetworkQueryService
	SourcingQuery *sourcingquery.SourcingQueryService
	RateQuery     *ratequery.RateQueryService
}
