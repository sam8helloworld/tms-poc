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
// 2. 既存の同一入札要求の契約を検索（重複チェック）
// 3. DRAFT契約を作成
// 4. 契約を永続化
// 5. 出力DTOの作成
//
// 想定される利用シーン:
// - 荷主が入札を開始し、複数の物流業者を選択した際に、各業者用のDRAFT契約を一括作成
// - 業者から入札参加の意思表示があった際に、個別にDRAFT契約を作成
type CreateBidContractUseCase struct {
	// 本来利用すべきリポジトリ（コメントアウト）
	// contractRepo contract.ServiceContractRepository
	// providerRepo contract.LogisticsProviderRepository
}

// NewCreateBidContractUseCase: CreateBidContractUseCaseのコンストラクタ
func NewCreateBidContractUseCase(
// contractRepo contract.ServiceContractRepository,
// providerRepo contract.LogisticsProviderRepository,
) *CreateBidContractUseCase {
	return &CreateBidContractUseCase{
		// contractRepo: contractRepo,
		// providerRepo: providerRepo,
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

	// 2. 物流業者の存在確認（コメントアウト）
	// provider, err := uc.providerRepo.FindByID(ctx, input.ProviderID)
	// if err != nil || provider == nil {
	// 	return nil, NewCreateBidContractError("PROVIDER_NOT_FOUND", "provider not found").
	// 		WithDetail("providerID", input.ProviderID)
	// }

	// 3. 既存の同一入札要求の契約を検索（重複チェック）
	// existingContracts, err := uc.contractRepo.FindDraftByProviderAndShipper(ctx, input.ProviderID, input.ShipperID)
	// if err != nil {
	// 	return nil, NewCreateBidContractError("REPO_ERROR", "failed to check existing contracts")
	// }
	// for _, existing := range existingContracts {
	// 	// 同じBidRequestIDの契約が既に存在する場合はエラー
	// 	if uc.hasSameBidRequest(existing, input.BidRequestID) {
	// 		return nil, NewCreateBidContractError("DUPLICATE_CONTRACT", "contract for this bid request already exists").
	// 			WithDetail("existingContractID", existing.ID)
	// 	}
	// }

	// 4. DRAFT契約を作成
	contract, err := contract.NewServiceContract(
		input.ProviderID,
		input.ShipperID,
		input.ValidFrom,
		input.ValidTo,
	)
	if err != nil {
		return nil, NewCreateBidContractError("CONTRACT_CREATE_ERROR", err.Error())
	}

	// 5. 契約を永続化（コメントアウト）
	// if err := uc.contractRepo.Save(ctx, contract); err != nil {
	// 	return nil, NewCreateBidContractError("SAVE_ERROR", "failed to save contract").
	// 		WithDetail("contractID", contract.ID)
	// }

	// 6. 出力DTOの作成
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

// hasSameBidRequest: 契約が同じBidRequestIDを持つかチェック
// 本来はServiceContractにBidRequestIDフィールドを追加するか、
// メタデータとして保存する必要がある
// func (uc *CreateBidContractUseCase) hasSameBidRequest(
// 	contract *contract.ServiceContract,
// 	bidRequestID uuid.UUID,
// ) bool {
// 	// 実装案: ServiceContractにBidRequestIDフィールドを追加
// 	// または、契約作成時のメタデータとして保存
// 	return false // プレースホルダー
// }
