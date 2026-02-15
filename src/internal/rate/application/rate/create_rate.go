package rate

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	domainrate "github.com/sam8helloworld/tms-poc/internal/rate/domain/rate"
)

// CreateRateUseCase: 新規レート作成ユースケース
// DRAFT状態の新しいレートを作成する
type CreateRateUseCase struct {
	rateRepo domainrate.RateRepository
}

// NewCreateRateUseCase: CreateRateUseCaseのコンストラクタ
func NewCreateRateUseCase(rateRepo domainrate.RateRepository) *CreateRateUseCase {
	return &CreateRateUseCase{rateRepo: rateRepo}
}

// CreateRateInput: レート作成の入力DTO
type CreateRateInput struct {
	ShipperID uuid.UUID
	Name      string
	ValidFrom time.Time
	ValidTo   time.Time
}

// CreateRateOutput: レート作成の出力DTO
type CreateRateOutput struct {
	RateID    uuid.UUID
	ShipperID uuid.UUID
	Name      string
	Status    string
	ValidFrom time.Time
	ValidTo   time.Time
	CreatedAt time.Time
}

// Execute: ユースケースを実行
func (uc *CreateRateUseCase) Execute(
	ctx context.Context,
	input CreateRateInput,
) (*CreateRateOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewCreateRateError("INVALID_INPUT", err.Error())
	}

	// 2. ドメインモデル生成
	r, err := domainrate.NewRate(input.ShipperID, input.Name, input.ValidFrom, input.ValidTo)
	if err != nil {
		return nil, NewCreateRateError("CREATE_ERROR", err.Error())
	}

	// 3. 永続化
	if err := uc.rateRepo.Save(ctx, r); err != nil {
		return nil, NewCreateRateError("SAVE_ERROR", "failed to save rate").
			WithDetail("rateID", r.ID)
	}

	return &CreateRateOutput{
		RateID:    r.ID,
		ShipperID: r.ShipperID,
		Name:      r.Name,
		Status:    string(r.Status()),
		ValidFrom: r.ValidPeriod.From,
		ValidTo:   r.ValidPeriod.To,
		CreatedAt: r.CreatedAt,
	}, nil
}

func (uc *CreateRateUseCase) validateInput(input CreateRateInput) error {
	if input.ShipperID == uuid.Nil {
		return errors.New("shipper ID is required")
	}
	if input.Name == "" {
		return errors.New("rate name is required")
	}
	if input.ValidFrom.After(input.ValidTo) {
		return errors.New("valid period is invalid: from must be before or equal to to")
	}
	return nil
}

// CreateRateError: レート作成時のエラー詳細
type CreateRateError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *CreateRateError) Error() string {
	return e.Message
}

func NewCreateRateError(code, message string) *CreateRateError {
	return &CreateRateError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *CreateRateError) WithDetail(key string, value any) *CreateRateError {
	e.Details[key] = value
	return e
}
