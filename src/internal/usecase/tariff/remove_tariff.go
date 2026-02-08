package tariff

import (
	"context"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/commercial"
)

// RemoveTariffInput: 料金表削除の入力
type RemoveTariffInput struct {
	ContractID uuid.UUID
	TariffID   uuid.UUID
}

// RemoveTariffOutput: 料金表削除の出力
type RemoveTariffOutput struct {
	ContractID         uuid.UUID
	ContractStatus     string
	RemovedTariffID    uuid.UUID
	RemovedTariffName  string
	RemainingTariffs   int
	Message            string
}

// RemoveTariffFromContractUseCase: 契約から料金表を削除するユースケース
// DRAFT状態の契約からのみ料金表を削除できる
type RemoveTariffFromContractUseCase struct {
	contractRepo commercial.ServiceContractRepository
	tariffRepo   commercial.TariffRepository
}

// NewRemoveTariffFromContractUseCase: コンストラクタ
func NewRemoveTariffFromContractUseCase(
	contractRepo commercial.ServiceContractRepository,
	tariffRepo commercial.TariffRepository,
) *RemoveTariffFromContractUseCase {
	return &RemoveTariffFromContractUseCase{
		contractRepo: contractRepo,
		tariffRepo:   tariffRepo,
	}
}

// Execute: ユースケースの実行
func (uc *RemoveTariffFromContractUseCase) Execute(
	ctx context.Context,
	input RemoveTariffInput,
) (*RemoveTariffOutput, error) {
	// 1. 契約を取得
	contract, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewRegisterTariffError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// DRAFT状態のみ料金表削除可能
	if !contract.IsDraft() {
		return nil, NewRegisterTariffError(
			"INVALID_CONTRACT_STATE",
			"tariffs can only be removed from DRAFT contracts",
		).WithDetail("currentStatus", string(contract.Status()))
	}

	// 2. 削除する料金表を取得
	tariff, err := uc.tariffRepo.FindByID(ctx, input.TariffID)
	if err != nil || tariff == nil {
		return nil, NewRegisterTariffError("TARIFF_NOT_FOUND", "tariff not found").
			WithDetail("tariffID", input.TariffID).
			WithDetail("contractID", input.ContractID)
	}
	if tariff.ContractID != input.ContractID {
		return nil, NewRegisterTariffError("TARIFF_NOT_IN_CONTRACT", "tariff does not belong to the specified contract").
			WithDetail("tariffID", input.TariffID).
			WithDetail("contractID", input.ContractID)
	}

	// 3. 料金表を削除
	if err := uc.tariffRepo.Delete(ctx, input.TariffID); err != nil {
		return nil, NewRegisterTariffError("REMOVE_ERROR", "failed to remove tariff").
			WithCause(err)
	}

	// 4. 残りのTariff件数を取得
	remainingCount, err := uc.tariffRepo.CountByContractID(ctx, input.ContractID)
	if err != nil {
		return nil, NewRegisterTariffError("COUNT_ERROR", "failed to count remaining tariffs").
			WithCause(err)
	}

	// 5. レスポンスを構築
	return &RemoveTariffOutput{
		ContractID:        contract.ID,
		ContractStatus:    string(contract.Status()),
		RemovedTariffID:   input.TariffID,
		RemovedTariffName: tariff.Name,
		RemainingTariffs:  remainingCount,
		Message:           "Tariff removed successfully from contract",
	}, nil
}
