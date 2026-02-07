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
}

// NewRemoveTariffFromContractUseCase: コンストラクタ
func NewRemoveTariffFromContractUseCase(
	contractRepo commercial.ServiceContractRepository,
) *RemoveTariffFromContractUseCase {
	return &RemoveTariffFromContractUseCase{
		contractRepo: contractRepo,
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

	// 2. 削除する料金表の情報を取得（レスポンス用）
	var removedTariffName string
	for _, tariff := range contract.Tariffs() {
		if tariff.ID == input.TariffID {
			removedTariffName = tariff.Name
			break
		}
	}
	if removedTariffName == "" {
		return nil, NewRegisterTariffError("TARIFF_NOT_FOUND", "tariff not found in contract").
			WithDetail("tariffID", input.TariffID).
			WithDetail("contractID", input.ContractID)
	}

	// 3. 料金表を削除
	if err := contract.RemoveTariff(input.TariffID); err != nil {
		return nil, NewRegisterTariffError("REMOVE_ERROR", "failed to remove tariff").
			WithCause(err)
	}

	// 4. 契約を保存
	if err := uc.contractRepo.Save(ctx, contract); err != nil {
		return nil, NewRegisterTariffError("SAVE_ERROR", "failed to save contract").
			WithCause(err)
	}

	// 5. レスポンスを構築
	return &RemoveTariffOutput{
		ContractID:        contract.ID,
		ContractStatus:    string(contract.Status()),
		RemovedTariffID:   input.TariffID,
		RemovedTariffName: removedTariffName,
		RemainingTariffs:  contract.TariffCount(),
		Message:           "Tariff removed successfully from contract",
	}, nil
}
