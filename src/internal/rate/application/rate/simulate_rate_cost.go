package rate

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// SimulateRateCostUseCase: レートコストシミュレーションユースケース
// ACTIVEレートに対して複数のルート条件・貨物条件で料金をシミュレーションする。
// TariffCalculator（ACLインターフェース）を通じてTariffの計算ロジックに委譲する。
type SimulateRateCostUseCase struct {
	rateRepo         domainrate.RateRepository
	tariffCalculator domainrate.TariffCalculator
}

// NewSimulateRateCostUseCase: SimulateRateCostUseCaseのコンストラクタ
func NewSimulateRateCostUseCase(
	rateRepo domainrate.RateRepository,
	tariffCalculator domainrate.TariffCalculator,
) *SimulateRateCostUseCase {
	return &SimulateRateCostUseCase{
		rateRepo:         rateRepo,
		tariffCalculator: tariffCalculator,
	}
}

// SimulateRateCostInput: シミュレーションの入力DTO
type SimulateRateCostInput struct {
	RateID          uuid.UUID              // 対象のレートID（ACTIVE状態であること）
	RouteConditions []SimulationRouteInput // シミュレーション対象のルート条件リスト
}

// SimulationRouteInput: 1ルートのシミュレーション入力
type SimulationRouteInput struct {
	Route    route.PhysicalRoute // 物理ルート（区間情報含む）
	Quantity decimal.Decimal     // 数量
	WeightKG decimal.Decimal     // 重量（kg）
	VolumeM3 decimal.Decimal     // 容積（m³）
}

// SimulateRateCostOutput: シミュレーション結果の出力DTO
type SimulateRateCostOutput struct {
	RateID       uuid.UUID
	RateName     string
	RateStatus   string
	RouteResults []RouteSimulationResult
}

// RouteSimulationResult: ルートごとのシミュレーション結果
type RouteSimulationResult struct {
	Route          route.PhysicalRoute
	AppliedCharges []SimulatedCharge  // 適用された費目（計算済み金額）
	SkippedCharges []SimulatedSkipped // スキップされた費目
	TotalAmounts   []CurrencyTotal    // 通貨別合計
	IsAvailable    bool               // 輸送可能か（適用費目が存在するか）
}

// SimulatedCharge: シミュレーションで適用された費目
type SimulatedCharge struct {
	EntryID    uuid.UUID
	ChargeCode string
	Category   string
	ProviderID uuid.UUID
	Amount     shared.Money // Tariff計算による実金額
}

// SimulatedSkipped: スキップされた費目
type SimulatedSkipped struct {
	EntryID    uuid.UUID
	ChargeCode string
	Category   string
	Reason     string
}

// CurrencyTotal: 通貨別合計
type CurrencyTotal struct {
	Currency string
	Amount   shared.Money
}

// Execute: ユースケースを実行
func (uc *SimulateRateCostUseCase) Execute(
	ctx context.Context,
	input SimulateRateCostInput,
) (*SimulateRateCostOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewSimulateRateCostError("INVALID_INPUT", err.Error())
	}

	// 2. レートを取得
	r, err := uc.rateRepo.FindByID(ctx, input.RateID)
	if err != nil {
		return nil, NewSimulateRateCostError("RATE_NOT_FOUND", "rate not found").
			WithDetail("rateID", input.RateID)
	}

	// 3. ACTIVE状態の確認
	if r.Status() != domainrate.RateStatusActive {
		return nil, NewSimulateRateCostError(
			"RATE_NOT_ACTIVE",
			"only ACTIVE rates can be used for simulation",
		).
			WithDetail("rateID", r.ID).
			WithDetail("status", string(r.Status()))
	}

	// 4. 各ルート条件に対してシミュレーション実行
	routeResults := make([]RouteSimulationResult, 0, len(input.RouteConditions))
	for _, routeInput := range input.RouteConditions {
		result, err := uc.simulateRoute(ctx, r, routeInput)
		if err != nil {
			return nil, err
		}
		routeResults = append(routeResults, *result)
	}

	return &SimulateRateCostOutput{
		RateID:       r.ID,
		RateName:     r.Name,
		RateStatus:   string(r.Status()),
		RouteResults: routeResults,
	}, nil
}

// simulateRoute: 1ルートに対するシミュレーションを実行
func (uc *SimulateRateCostUseCase) simulateRoute(
	ctx context.Context,
	r *domainrate.Rate,
	routeInput SimulationRouteInput,
) (*RouteSimulationResult, error) {
	// ルートの最初のセグメントからOrigin/Destination/Modeを取得
	if len(routeInput.Route.Segments) == 0 {
		return &RouteSimulationResult{
			Route:          routeInput.Route,
			AppliedCharges: make([]SimulatedCharge, 0),
			SkippedCharges: make([]SimulatedSkipped, 0),
			TotalAmounts:   make([]CurrencyTotal, 0),
			IsAvailable:    false,
		}, nil
	}

	// ドメインモデルのルート検索を利用して該当エントリを検索
	matched := r.FindEntriesForRoute(
		routeInput.Route.OriginID,
		routeInput.Route.DestinationID,
		routeInput.Route.Segments[0].Mode,
	)

	if len(matched) == 0 {
		return &RouteSimulationResult{
			Route:          routeInput.Route,
			AppliedCharges: make([]SimulatedCharge, 0),
			SkippedCharges: make([]SimulatedSkipped, 0),
			TotalAmounts:   make([]CurrencyTotal, 0),
			IsAvailable:    false,
		}, nil
	}

	// 各RateEntryに対してTariffCalculatorで料金計算
	appliedCharges := make([]SimulatedCharge, 0, len(matched))
	skippedCharges := make([]SimulatedSkipped, 0)
	totals := make(map[string]shared.Money)

	for _, entry := range matched {
		calcInput := domainrate.TariffCalculationInput{
			TariffID:         entry.TariffID,
			TariffLineItemID: entry.TariffLineItemID,
			Route:            routeInput.Route,
			Quantity:         routeInput.Quantity,
			WeightKG:         routeInput.WeightKG,
			VolumeM3:         routeInput.VolumeM3,
		}

		chargeResult, err := uc.tariffCalculator.Calculate(ctx, calcInput)
		if err != nil {
			return nil, NewSimulateRateCostError("CALCULATION_ERROR",
				"tariff calculation failed for entry "+entry.ID.String()).
				WithDetail("entryID", entry.ID).
				WithDetail("tariffID", entry.TariffID)
		}

		if chargeResult.Skipped {
			skippedCharges = append(skippedCharges, SimulatedSkipped{
				EntryID:    entry.ID,
				ChargeCode: chargeResult.ChargeCode,
				Category:   chargeResult.Category,
				Reason:     chargeResult.SkipReason,
			})
			continue
		}

		appliedCharges = append(appliedCharges, SimulatedCharge{
			EntryID:    entry.ID,
			ChargeCode: chargeResult.ChargeCode,
			Category:   chargeResult.Category,
			ProviderID: entry.ProviderID,
			Amount:     chargeResult.Amount,
		})

		// 通貨別合計を更新
		currency := chargeResult.Amount.Currency
		if existing, ok := totals[currency]; ok {
			sum, addErr := existing.Add(chargeResult.Amount)
			if addErr == nil {
				totals[currency] = sum
			}
		} else {
			totals[currency] = chargeResult.Amount
		}
	}

	// 通貨別合計をスライスに変換
	totalAmounts := make([]CurrencyTotal, 0, len(totals))
	for currency, amount := range totals {
		totalAmounts = append(totalAmounts, CurrencyTotal{
			Currency: currency,
			Amount:   amount,
		})
	}

	return &RouteSimulationResult{
		Route:          routeInput.Route,
		AppliedCharges: appliedCharges,
		SkippedCharges: skippedCharges,
		TotalAmounts:   totalAmounts,
		IsAvailable:    len(appliedCharges) > 0,
	}, nil
}

func (uc *SimulateRateCostUseCase) validateInput(input SimulateRateCostInput) error {
	if input.RateID == uuid.Nil {
		return errors.New("rateID is required")
	}
	if len(input.RouteConditions) == 0 {
		return errors.New("at least one route condition is required")
	}
	for _, cond := range input.RouteConditions {
		if len(cond.Route.Segments) == 0 {
			return errors.New("route must have at least one segment")
		}
		if cond.Quantity.LessThanOrEqual(decimal.Zero) {
			return errors.New("quantity must be positive")
		}
	}
	return nil
}

// SimulateRateCostError: シミュレーション時のエラー詳細
type SimulateRateCostError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *SimulateRateCostError) Error() string {
	return e.Message
}

func NewSimulateRateCostError(code, message string) *SimulateRateCostError {
	return &SimulateRateCostError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *SimulateRateCostError) WithDetail(key string, value any) *SimulateRateCostError {
	e.Details[key] = value
	return e
}
