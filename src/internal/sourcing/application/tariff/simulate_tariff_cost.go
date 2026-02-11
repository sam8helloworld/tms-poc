package tariff

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// SimulateTariffCostInput: 料金シミュレーションの入力DTO
type SimulateTariffCostInput struct {
	TariffID   uuid.UUID
	Route      route.PhysicalRoute
	CargoItems []pricing.CargoItem     // 貨物明細（Summary自動計算）
	Summary    *pricing.CargoSummary   // 概算用に直接指定する場合
	Conditions pricing.CalculationConditions
}

// SimulateTariffCostOutput: 料金シミュレーションの出力DTO
type SimulateTariffCostOutput struct {
	TariffID      uuid.UUID
	TariffName    string
	TariffVersion int
	Result        *pricing.TariffCalculationResult
	SimulatedAt   time.Time
}

// SimulateTariffCostError: 料金シミュレーション時のエラー詳細
type SimulateTariffCostError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *SimulateTariffCostError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *SimulateTariffCostError) Unwrap() error {
	return e.Cause
}

func NewSimulateTariffCostError(code, message string) *SimulateTariffCostError {
	return &SimulateTariffCostError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *SimulateTariffCostError) WithDetail(key string, value any) *SimulateTariffCostError {
	e.Details[key] = value
	return e
}

func (e *SimulateTariffCostError) WithCause(cause error) *SimulateTariffCostError {
	e.Cause = cause
	return e
}

// SimulateTariffCostUseCase: 料金表コストシミュレーションユースケース
type SimulateTariffCostUseCase struct {
	tariffRepo pricing.TariffRepository
}

// NewSimulateTariffCostUseCase: コンストラクタ
func NewSimulateTariffCostUseCase(tariffRepo pricing.TariffRepository) *SimulateTariffCostUseCase {
	return &SimulateTariffCostUseCase{
		tariffRepo: tariffRepo,
	}
}

// Execute: 料金シミュレーションを実行
func (uc *SimulateTariffCostUseCase) Execute(
	ctx context.Context,
	input SimulateTariffCostInput,
) (*SimulateTariffCostOutput, error) {
	// 1. 入力バリデーション
	if input.TariffID == uuid.Nil {
		return nil, NewSimulateTariffCostError("INVALID_INPUT", "tariff ID is required")
	}
	if len(input.CargoItems) == 0 && input.Summary == nil {
		return nil, NewSimulateTariffCostError("INVALID_INPUT", "either cargo items or summary is required")
	}

	// 2. Tariff取得
	tariff, err := uc.tariffRepo.FindByID(ctx, input.TariffID)
	if err != nil {
		return nil, NewSimulateTariffCostError("TARIFF_NOT_FOUND", "failed to find tariff").
			WithDetail("tariffID", input.TariffID.String()).
			WithCause(err)
	}
	if tariff == nil {
		return nil, NewSimulateTariffCostError("TARIFF_NOT_FOUND", "tariff does not exist").
			WithDetail("tariffID", input.TariffID.String())
	}

	// 3. CalculationRequest構築
	var calcReq *pricing.CalculationRequest
	if input.Summary != nil {
		calcReq, err = pricing.NewCalculationRequestWithSummary(input.Route, *input.Summary, input.Conditions)
	} else {
		calcReq, err = pricing.NewCalculationRequest(input.Route, input.CargoItems, input.Conditions)
	}
	if err != nil {
		return nil, NewSimulateTariffCostError("INVALID_INPUT", "failed to build calculation request").
			WithCause(err)
	}

	// 4. コスト計算
	result, err := tariff.CalculateCharges(*calcReq)
	if err != nil {
		return nil, NewSimulateTariffCostError("CALCULATION_ERROR", "charge calculation failed").
			WithDetail("tariffID", input.TariffID.String()).
			WithCause(err)
	}

	// 5. 出力
	return &SimulateTariffCostOutput{
		TariffID:      tariff.ID,
		TariffName:    tariff.Name,
		TariffVersion: tariff.Version,
		Result:        result,
		SimulatedAt:   time.Now(),
	}, nil
}
