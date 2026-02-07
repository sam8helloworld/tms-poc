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
// 入札プロセスにおいて、物流企業から提示された料金表を含む契約情報
// DRAFT状態で作成され、料金表を登録・比較し、最終的にCONTRACTED状態になる
type ServiceContract struct {
	shared.EventRecorder

	ID          uuid.UUID
	ProviderID  uuid.UUID      // 物流企業ID
	ShipperID   uuid.UUID      // 荷主企業ID
	status      ContractStatus // 契約ステータス
	ValidPeriod shared.DateRange
	tariffs     []*Tariff

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Status: ステータスのgetter
func (c *ServiceContract) Status() ContractStatus {
	return c.status
}

// Tariffs: 料金表のgetter（コピーを返却）
func (c *ServiceContract) Tariffs() []*Tariff {
	result := make([]*Tariff, len(c.tariffs))
	copy(result, c.tariffs)
	return result
}

// TariffCount: 料金表の数を返す
func (c *ServiceContract) TariffCount() int {
	return len(c.tariffs)
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
		tariffs:   make([]*Tariff, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// AddOrUpdateTariff: 料金表を追加または更新
// 同じID（UUID）のTariffがあれば更新、なければ追加
// バージョン管理: 同じ名前でもバージョンが異なれば別レコードとして追加される
func (c *ServiceContract) AddOrUpdateTariff(tariff *Tariff) error {
	if err := c.canModifyTariffs(); err != nil {
		return err
	}

	if tariff == nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "tariff is required")
	}

	// バリデーション
	if err := tariff.Validate(); err != nil {
		return err
	}

	// 既存のTariffを検索（IDで判定）
	for i, existing := range c.tariffs {
		if existing.ID == tariff.ID {
			// 更新（同じIDの場合のみ）
			c.tariffs[i] = tariff
			c.UpdatedAt = time.Now()
			c.RecordEvent(NewTariffRegistered(c.ID, tariff.ID, tariff.Name, true))
			return nil
		}
	}

	// 追加（新規または新バージョン）
	c.tariffs = append(c.tariffs, tariff)
	c.UpdatedAt = time.Now()
	isUpdate := tariff.IsNewVersion() // 新バージョンの場合はtrueになる
	c.RecordEvent(NewTariffRegistered(c.ID, tariff.ID, tariff.Name, isUpdate))
	return nil
}

// RemoveTariff: 料金表を削除
func (c *ServiceContract) RemoveTariff(tariffID uuid.UUID) error {
	if err := c.canModifyTariffs(); err != nil {
		return err
	}

	for i, tariff := range c.tariffs {
		if tariff.ID == tariffID {
			c.tariffs = append(c.tariffs[:i], c.tariffs[i+1:]...)
			c.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError(shared.ErrNotFound, "tariff not found")
}

// MarkAsContracted: 契約を成立させる
// 入札プロセスを経て、この契約を正式なものとする
func (c *ServiceContract) MarkAsContracted() error {
	if c.status != ContractStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT contracts can be marked as CONTRACTED")
	}
	if len(c.tariffs) == 0 {
		return shared.NewDomainError(shared.ErrBusinessRuleViolation, "contract must have at least one tariff to be contracted")
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

	// CONTRACTED状態では少なくとも1つのTariffが必要
	if c.status == ContractStatusContracted && len(c.tariffs) == 0 {
		return shared.NewDomainError(shared.ErrBusinessRuleViolation, "contracted contract must have at least one tariff")
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

// canModifyTariffs: Tariffの追加・更新・削除が可能な状態か確認
func (c *ServiceContract) canModifyTariffs() error {
	if c.status != ContractStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "tariffs can only be modified in DRAFT status")
	}
	return nil
}

// FindTariffsByName: 指定された名前のすべてのTariffバージョンを取得
func (c *ServiceContract) FindTariffsByName(name string) []*Tariff {
	var result []*Tariff
	for _, tariff := range c.tariffs {
		if tariff.Name == name {
			result = append(result, tariff)
		}
	}
	return result
}

// FindLatestTariffVersion: 指定された名前の最新バージョンのTariffを取得
func (c *ServiceContract) FindLatestTariffVersion(name string) *Tariff {
	var latest *Tariff
	for _, tariff := range c.tariffs {
		if tariff.Name == name {
			if latest == nil || tariff.Version > latest.Version {
				latest = tariff
			}
		}
	}
	return latest
}
