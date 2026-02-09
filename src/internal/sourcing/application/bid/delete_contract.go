package bid

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
)

// DeleteBidContractInput: 入札契約削除の入力
type DeleteBidContractInput struct {
	ContractID uuid.UUID
}

// DeleteBidContractOutput: 入札契約削除の出力
type DeleteBidContractOutput struct {
	ContractID uuid.UUID
	ProviderID uuid.UUID
	ShipperID  uuid.UUID
	Status     string
	DeletedAt  time.Time
}

// DeleteBidContractUseCase: 入札契約を削除（キャンセル）するユースケース
// DRAFT状態の契約をCANCELLED状態にする（論理削除）
type DeleteBidContractUseCase struct {
	contractRepo contract.ServiceContractRepository
}

// NewDeleteBidContractUseCase: コンストラクタ
func NewDeleteBidContractUseCase(
	contractRepo contract.ServiceContractRepository,
) *DeleteBidContractUseCase {
	return &DeleteBidContractUseCase{
		contractRepo: contractRepo,
	}
}

// Execute: ユースケースの実行
func (uc *DeleteBidContractUseCase) Execute(
	ctx context.Context,
	input DeleteBidContractInput,
) (*DeleteBidContractOutput, error) {
	// 契約を取得
	contract, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewCreateBidContractError("NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// DRAFT状態のみ削除可能
	if !contract.IsDraft() {
		return nil, NewCreateBidContractError(
			"INVALID_STATE",
			"only DRAFT contracts can be deleted",
		).WithDetail("currentStatus", string(contract.Status()))
	}

	// 契約をキャンセル状態にする（論理削除）
	if err := contract.MarkAsCancelled(); err != nil {
		return nil, NewCreateBidContractError("CANCEL_ERROR", "failed to cancel contract").
			WithDetail("contractID", contract.ID)
	}

	deletedAt := time.Now()

	// 保存
	if err := uc.contractRepo.Save(ctx, contract); err != nil {
		return nil, NewCreateBidContractError("SAVE_ERROR", "failed to delete bid contract").
			WithDetail("contractID", contract.ID)
	}

	// レスポンスを構築
	return &DeleteBidContractOutput{
		ContractID: contract.ID,
		ProviderID: contract.ProviderID,
		ShipperID:  contract.ShipperID,
		Status:     string(contract.Status()),
		DeletedAt:  deletedAt,
	}, nil
}
