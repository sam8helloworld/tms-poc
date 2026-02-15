package bid

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/contract"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// AwardBidContractUseCase: 入札契約を成立させるユースケース
// DRAFT状態の契約をCONTRACTED状態に遷移させる
type AwardBidContractUseCase struct {
	contractRepo contract.ServiceContractRepository
	tariffRepo   pricing.TariffRepository
}

// NewAwardBidContractUseCase: AwardBidContractUseCaseのコンストラクタ
func NewAwardBidContractUseCase(
	contractRepo contract.ServiceContractRepository,
	tariffRepo pricing.TariffRepository,
) *AwardBidContractUseCase {
	return &AwardBidContractUseCase{
		contractRepo: contractRepo,
		tariffRepo:   tariffRepo,
	}
}

// AwardBidContractInput: 入札契約成立の入力DTO
type AwardBidContractInput struct {
	ContractID uuid.UUID
}

// AwardBidContractOutput: 入札契約成立の出力DTO
type AwardBidContractOutput struct {
	ContractID   uuid.UUID
	ProviderID   uuid.UUID
	Status       string
	TariffCount  int
	AwardedAt    time.Time
}

// Execute: ユースケースを実行
func (uc *AwardBidContractUseCase) Execute(
	ctx context.Context,
	input AwardBidContractInput,
) (*AwardBidContractOutput, error) {
	// 1. 入力バリデーション
	if input.ContractID == uuid.Nil {
		return nil, NewCreateBidContractError("INVALID_INPUT", "contract ID is required")
	}

	// 2. 契約の取得
	c, err := uc.contractRepo.FindByID(ctx, input.ContractID)
	if err != nil {
		return nil, NewCreateBidContractError("CONTRACT_NOT_FOUND", "contract not found").
			WithDetail("contractID", input.ContractID)
	}

	// 3. DRAFT状態チェック
	if !c.IsDraft() {
		return nil, NewCreateBidContractError("CONTRACT_NOT_DRAFT", "only DRAFT contracts can be awarded").
			WithDetail("contractID", input.ContractID).
			WithDetail("status", string(c.Status()))
	}

	// 4. Tariffが1件以上存在するかチェック
	tariffCount, err := uc.tariffRepo.CountByContractID(ctx, input.ContractID)
	if err != nil {
		return nil, NewCreateBidContractError("TARIFF_COUNT_ERROR", "failed to count tariffs").
			WithDetail("contractID", input.ContractID)
	}
	if tariffCount == 0 {
		return nil, NewCreateBidContractError("NO_TARIFFS", "contract must have at least one tariff to be awarded").
			WithDetail("contractID", input.ContractID)
	}

	// 5. DRAFT → CONTRACTED
	if err := c.MarkAsContracted(); err != nil {
		return nil, NewCreateBidContractError("STATE_TRANSITION_ERROR", err.Error()).
			WithDetail("contractID", input.ContractID)
	}

	// 6. 永続化
	if err := uc.contractRepo.Save(ctx, c); err != nil {
		return nil, NewCreateBidContractError("SAVE_ERROR", "failed to save contract").
			WithDetail("contractID", input.ContractID)
	}

	return &AwardBidContractOutput{
		ContractID:  c.ID,
		ProviderID:  c.ProviderID,
		Status:      string(c.Status()),
		TariffCount: tariffCount,
		AwardedAt:   time.Now(),
	}, nil
}
