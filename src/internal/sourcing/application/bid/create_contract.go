package bid

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
)

// CreateBidContractUseCase: 入札契約作成ユースケース
// 入札プロセスにおいて、各物流業者との契約をDRAFT状態で作成する
//
// 処理の流れ:
// 1. 入力バリデーション
// 2. DRAFT契約を作成
// 3. 契約を永続化
// 4. 出力DTOの作成
type CreateBidContractUseCase struct {
	contractRepo contract.ServiceContractRepository
}

// NewCreateBidContractUseCase: CreateBidContractUseCaseのコンストラクタ
func NewCreateBidContractUseCase(
	contractRepo contract.ServiceContractRepository,
) *CreateBidContractUseCase {
	return &CreateBidContractUseCase{
		contractRepo: contractRepo,
	}
}

// Execute: ユースケースを実行
func (uc *CreateBidContractUseCase) Execute(
	ctx context.Context,
	input CreateBidContractInput,
) (*CreateBidContractOutput, error) {
	// 1. 入力バリデーション
	if err := uc.validateInput(input); err != nil {
		return nil, NewCreateBidContractError("INVALID_INPUT", err.Error())
	}

	// 2. DRAFT契約を作成
	contract, err := contract.NewServiceContract(
		input.ProviderID,
		input.ShipperID,
		input.ValidFrom,
		input.ValidTo,
	)
	if err != nil {
		return nil, NewCreateBidContractError("CONTRACT_CREATE_ERROR", err.Error())
	}

	// 3. 契約を永続化
	if err := uc.contractRepo.Save(ctx, contract); err != nil {
		return nil, NewCreateBidContractError("SAVE_ERROR", "failed to save contract").
			WithDetail("contractID", contract.ID)
	}

	// 4. 出力DTOの作成
	output := &CreateBidContractOutput{
		ContractID:      contract.ID,
		ProviderID:      contract.ProviderID,
		ShipperID:       contract.ShipperID,
		Status:          string(contract.Status()),
		ValidFrom:       contract.ValidPeriod.From,
		ValidTo:         contract.ValidPeriod.To,
		CreatedAt:       contract.CreatedAt,
		BidRequestID:    input.BidRequestID,
		BidRequestName:  input.BidRequestName,
		TargetRouteInfo: input.TargetRoutes,
		NextSteps: []string{
			"物流業者からの料金表提出を待つ",
			fmt.Sprintf("料金表が届いたらRegisterTariffUseCaseで契約ID %s に登録", contract.ID),
			"すべての料金表が揃ったら各契約を比較検討",
			"最適な契約をMarkAsContractedで正式契約化",
		},
	}

	return output, nil
}

// validateInput: 入力の基本的なバリデーション
func (uc *CreateBidContractUseCase) validateInput(input CreateBidContractInput) error {
	if input.BidRequestID == uuid.Nil {
		return errors.New("bid request ID is required")
	}
	if input.ProviderID == uuid.Nil {
		return errors.New("provider ID is required")
	}
	if input.ShipperID == uuid.Nil {
		return errors.New("shipper ID is required")
	}
	if input.ValidFrom.IsZero() {
		return errors.New("valid from date is required")
	}
	if input.ValidTo.IsZero() {
		return errors.New("valid to date is required")
	}
	if input.ValidFrom.After(input.ValidTo) {
		return errors.New("valid from must be before or equal to valid to")
	}

	return nil
}
