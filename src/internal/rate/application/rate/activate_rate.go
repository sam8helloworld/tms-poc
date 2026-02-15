package rate

import (
	"context"
	"time"

	"github.com/google/uuid"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
)

// ActivateRateUseCase: レートを有効化するユースケース
// DRAFT状態のレートをACTIVE状態に遷移させる
type ActivateRateUseCase struct {
	rateRepo domainrate.RateRepository
}

// NewActivateRateUseCase: ActivateRateUseCaseのコンストラクタ
func NewActivateRateUseCase(rateRepo domainrate.RateRepository) *ActivateRateUseCase {
	return &ActivateRateUseCase{rateRepo: rateRepo}
}

// ActivateRateInput: レート有効化の入力DTO
type ActivateRateInput struct {
	RateID uuid.UUID
}

// ActivateRateOutput: レート有効化の出力DTO
type ActivateRateOutput struct {
	RateID      uuid.UUID
	Status      string
	EntryCount  int
	ActivatedAt time.Time
}

// Execute: ユースケースを実行
func (uc *ActivateRateUseCase) Execute(
	ctx context.Context,
	input ActivateRateInput,
) (*ActivateRateOutput, error) {
	// 1. 入力バリデーション
	if input.RateID == uuid.Nil {
		return nil, NewActivateRateError("INVALID_INPUT", "rate ID is required")
	}

	// 2. レートの取得
	r, err := uc.rateRepo.FindByID(ctx, input.RateID)
	if err != nil {
		return nil, NewActivateRateError("RATE_NOT_FOUND", "rate not found").
			WithDetail("rateID", input.RateID)
	}

	// 3. DRAFT → ACTIVE（ドメインメソッドでステータス・エントリ数チェック）
	if err := r.Activate(); err != nil {
		return nil, NewActivateRateError("ACTIVATION_ERROR", err.Error()).
			WithDetail("rateID", input.RateID).
			WithDetail("status", string(r.Status()))
	}

	// 4. 永続化
	if err := uc.rateRepo.Save(ctx, r); err != nil {
		return nil, NewActivateRateError("SAVE_ERROR", "failed to save rate").
			WithDetail("rateID", input.RateID)
	}

	return &ActivateRateOutput{
		RateID:      r.ID,
		Status:      string(r.Status()),
		EntryCount:  len(r.Entries()),
		ActivatedAt: time.Now(),
	}, nil
}

// ActivateRateError: レート有効化時のエラー詳細
type ActivateRateError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *ActivateRateError) Error() string {
	return e.Message
}

func NewActivateRateError(code, message string) *ActivateRateError {
	return &ActivateRateError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *ActivateRateError) WithDetail(key string, value any) *ActivateRateError {
	e.Details[key] = value
	return e
}
