package commercial

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ContractStatus: 契約のステータス
type ContractStatus string

const (
	// ContractStatusDraft: 入札段階（ドラフト）
	// 各社から料金表を受け取り、比較検討する段階
	ContractStatusDraft ContractStatus = "DRAFT"

	// ContractStatusContracted: 契約成立
	// 入札を経て正式に契約が成立した状態
	ContractStatusContracted ContractStatus = "CONTRACTED"

	// ContractStatusExpired: 期限切れ
	// 契約期間が終了した状態
	ContractStatusExpired ContractStatus = "EXPIRED"

	// ContractStatusCancelled: キャンセル
	// 契約が破棄された状態
	ContractStatusCancelled ContractStatus = "CANCELLED"
)

// ServiceContract: 契約（集約ルート）
// 入札プロセスにおいて、物流企業から提示された契約情報を管理する
// DRAFT状態で作成され、最終的にCONTRACTED状態になる
// 料金表(Tariff)は独立した集約として管理され、ContractIDで参照される
type ServiceContract struct {
	shared.EventRecorder

	ID          uuid.UUID
	ProviderID  uuid.UUID      // 物流企業ID
	ShipperID   uuid.UUID      // 荷主企業ID
	status      ContractStatus // 契約ステータス
	ValidPeriod shared.DateRange

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Status: ステータスのgetter
func (c *ServiceContract) Status() ContractStatus {
	return c.status
}

// NewServiceContract: ServiceContractのファクトリー関数
// DRAFT状態で新しい契約を作成
func NewServiceContract(
	providerID uuid.UUID,
	shipperID uuid.UUID,
	validFrom time.Time,
	validTo time.Time,
) (*ServiceContract, error) {
	if providerID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "providerID is required")
	}
	if shipperID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipperID is required")
	}
	if validFrom.After(validTo) {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "valid period is invalid: from must be before or equal to to")
	}

	now := time.Now()
	return &ServiceContract{
		ID:         uuid.New(),
		ProviderID: providerID,
		ShipperID:  shipperID,
		status:     ContractStatusDraft, // 初期状態はDRAFT
		ValidPeriod: shared.DateRange{
			From: validFrom,
			To:   validTo,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// MarkAsContracted: 契約を成立させる
// 入札プロセスを経て、この契約を正式なものとする
// Tariffの存在チェックはUseCase層で実施する
func (c *ServiceContract) MarkAsContracted() error {
	if c.status != ContractStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT contracts can be marked as CONTRACTED")
	}

	oldStatus := c.status
	c.status = ContractStatusContracted
	c.UpdatedAt = time.Now()
	c.RecordEvent(NewContractStatusChanged(c.ID, oldStatus, c.status))
	return nil
}

// MarkAsExpired: 契約を期限切れにする
func (c *ServiceContract) MarkAsExpired() error {
	if c.status != ContractStatusContracted {
		return shared.NewDomainError(shared.ErrInvalidState, "only CONTRACTED contracts can be marked as EXPIRED")
	}

	oldStatus := c.status
	c.status = ContractStatusExpired
	c.UpdatedAt = time.Now()
	c.RecordEvent(NewContractStatusChanged(c.ID, oldStatus, c.status))
	return nil
}

// MarkAsCancelled: 契約をキャンセル
func (c *ServiceContract) MarkAsCancelled() error {
	if c.status == ContractStatusExpired || c.status == ContractStatusCancelled {
		return shared.NewDomainError(shared.ErrInvalidState, "cannot cancel an expired or already cancelled contract")
	}

	oldStatus := c.status
	c.status = ContractStatusCancelled
	c.UpdatedAt = time.Now()
	c.RecordEvent(NewContractStatusChanged(c.ID, oldStatus, c.status))
	return nil
}

// Validate: 契約のビジネスルールを検証
func (c *ServiceContract) Validate() error {
	if c.ID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "contract ID is required")
	}
	if c.ProviderID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "provider ID is required")
	}
	if c.ShipperID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "shipper ID is required")
	}
	if c.ValidPeriod.From.After(c.ValidPeriod.To) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "valid period is invalid")
	}

	return nil
}

// IsActive: 契約が有効（使用可能）かどうか判定
func (c *ServiceContract) IsActive() bool {
	return c.status == ContractStatusContracted
}

// IsDraft: ドラフト状態かどうか判定
func (c *ServiceContract) IsDraft() bool {
	return c.status == ContractStatusDraft
}

