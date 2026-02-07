package bid

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// UpdateContractPeriodInput: 契約期間更新の入力
type UpdateContractPeriodInput struct {
	ContractID uuid.UUID
	ValidFrom  time.Time
	ValidTo    time.Time
}

// UpdateContractPeriodOutput: 契約期間更新の出力
type UpdateContractPeriodOutput struct {
	ContractID  uuid.UUID
	ProviderID  uuid.UUID
	ShipperID   uuid.UUID
	Status      string
	ValidFrom   time.Time
	ValidTo     time.Time
	UpdatedAt   time.Time
	TariffCount int
}

// UpdateContractPeriodUseCase: 契約期間を更新するユースケース
// DRAFT状態の契約の有効期間（ValidPeriod）を変更する
type UpdateContractPeriodUseCase struct {
	contractRepo commercial.ServiceContractRepository
}

// NewUpdateContractPeriodUseCase: コンストラクタ
func NewUpdateContractPeriodUseCase(
	contractRepo commercial.ServiceContractRepository,
) *UpdateContractPeriodUseCase {
	return &UpdateContractPeriodUseCase{
		contractRepo: contractRepo,
	}
}

// Execute: ユースケースの実行
func (uc *UpdateContractPeriodUseCase) Execute(
	ctx context.Context,
	input UpdateContractPeriodInput,
) (*UpdateContractPeriodOutput, error) {
	// 期間のバリデーション
	if input.ValidFrom.After(input.ValidTo) {
		return nil, NewCreateBidContractError(
			"INVALID_INPUT",
			"valid period is invalid: from must be before or equal to to",
		)
	}

	// 契約を取得
	contract, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewCreateBidContractError("NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// DRAFT状態のみ更新可能
	if !contract.IsDraft() {
		return nil, NewCreateBidContractError(
			"INVALID_STATE",
			"only DRAFT contracts can have their period updated",
		).WithDetail("currentStatus", string(contract.Status()))
	}

	// 契約期間を更新（直接フィールドを更新）
	contract.ValidPeriod = shared.DateRange{
		From: input.ValidFrom,
		To:   input.ValidTo,
	}
	contract.UpdatedAt = time.Now()

	// 保存
	if err := uc.contractRepo.Save(ctx, contract); err != nil {
		return nil, NewCreateBidContractError(
			"SAVE_ERROR",
			"failed to update contract period",
		).WithDetail("contractID", contract.ID)
	}

	// レスポンスを構築
	return &UpdateContractPeriodOutput{
		ContractID:  contract.ID,
		ProviderID:  contract.ProviderID,
		ShipperID:   contract.ShipperID,
		Status:      string(contract.Status()),
		ValidFrom:   contract.ValidPeriod.From,
		ValidTo:     contract.ValidPeriod.To,
		UpdatedAt:   contract.UpdatedAt,
		TariffCount: contract.TariffCount(),
	}, nil
}
