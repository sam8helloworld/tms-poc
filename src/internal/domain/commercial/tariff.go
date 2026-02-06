package commercial

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/pricing"
	"github.com/sam8helloworld/tms-poc/internal/domain/logic/scope"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// Tariff: 料金表 (Rate Book / Catalog)
// 1つの契約の下にぶら下がる、大量の料金項目の集合体
type Tariff struct {
	ID            uuid.UUID
	ContractID    uuid.UUID
	Name          string // e.g. "2026 Japan Export"
	EffectiveDate shared.DateRange

	// LineItems: 個別の料金定義のリスト
	LineItems []TariffLineItem
}

// NewTariff: Tariffのファクトリー関数
// ドメイン不変条件を保証した状態でTariffを生成する
func NewTariff(
	contractID uuid.UUID,
	name string,
	effectiveFrom time.Time,
	effectiveTo time.Time,
) (*Tariff, error) {
	if contractID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "contractID is required")
	}
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "tariff name is required")
	}
	if effectiveFrom.After(effectiveTo) {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "effective date range is invalid: from must be before or equal to to")
	}

	return &Tariff{
		ID:         uuid.New(),
		ContractID: contractID,
		Name:       name,
		EffectiveDate: shared.DateRange{
			From: effectiveFrom,
			To:   effectiveTo,
		},
		LineItems: make([]TariffLineItem, 0),
	}, nil
}

// AddLineItem: 料金明細を追加する
// 集約の整合性を保つため、直接LineItemsを操作せずこのメソッドを使用する
func (t *Tariff) AddLineItem(item TariffLineItem) error {
	if item.ChargeCode == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "charge code is required")
	}
	if item.Scope == nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "service scope is required")
	}
	if item.Logic == nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "pricing logic is required")
	}

	// IDが未設定の場合は自動生成
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}

	t.LineItems = append(t.LineItems, item)
	return nil
}

// Validate: Tariffのビジネスルールを検証
func (t *Tariff) Validate() error {
	if t.ID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "tariff ID is required")
	}
	if t.ContractID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "contract ID is required")
	}
	if t.Name == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "tariff name is required")
	}
	if t.EffectiveDate.From.After(t.EffectiveDate.To) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "effective date range is invalid")
	}
	if len(t.LineItems) == 0 {
		return shared.NewDomainError(shared.ErrBusinessRuleViolation, "tariff must have at least one line item")
	}

	// 各LineItemのバリデーション
	for i, item := range t.LineItems {
		if item.ChargeCode == "" {
			return shared.NewDomainError(shared.ErrInvalidArgument, "line item at index "+strconv.Itoa(i)+" has empty charge code")
		}
		if item.Scope == nil {
			return shared.NewDomainError(shared.ErrInvalidArgument, "line item at index "+strconv.Itoa(i)+" has no service scope")
		}
		if item.Logic == nil {
			return shared.NewDomainError(shared.ErrInvalidArgument, "line item at index "+strconv.Itoa(i)+" has no pricing logic")
		}
	}

	return nil
}

// IsEffectiveAt: 指定日時にこのTariffが有効かどうか判定
func (t *Tariff) IsEffectiveAt(date time.Time) bool {
	return !date.Before(t.EffectiveDate.From) && !date.After(t.EffectiveDate.To)
}

// TariffLineItem: 1行の料金定義 (The Rate)
type TariffLineItem struct {
	ID         uuid.UUID
	ChargeCode string // "OFT", "THC"
	Category   string // FREIGHT, LOCAL

	// Scope: どこに適用されるか (ドメイン用語で定義)
	Scope scope.ServiceScope

	// Logic: いくらか (Strategy Pattern)
	Logic pricing.PricingStrategy
}
