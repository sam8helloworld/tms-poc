package rate

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ApplyContractToRateInput: 契約反映ユースケースの入力DTO
// CONTRACTED状態の契約から料金表（一部または全部）を選択し、レートに反映する
type ApplyContractToRateInput struct {
	RateID     uuid.UUID // 反映先のレートID（DRAFT状態であること）
	ContractID uuid.UUID // 反映元の契約ID（CONTRACTED状態であること）

	// 反映する料金表のID（空の場合は契約内の全Tariffを反映）
	TariffIDs []uuid.UUID

	// 作成される全RateEntryに適用するルート範囲
	RouteScope RouteScopeInput
}

// RouteScopeInput: ルート範囲の入力DTO
type RouteScopeInput struct {
	OriginID      *route.LocationID    // nil = 全Origin
	DestinationID *route.LocationID    // nil = 全Destination
	TransportMode *shared.TransportMode // nil = 全モード
}

// ApplyContractToRateOutput: 契約反映ユースケースの出力DTO
type ApplyContractToRateOutput struct {
	RateID          uuid.UUID
	RateStatus      string
	ContractID      uuid.UUID
	ProviderID      uuid.UUID
	AddedEntries    []AddedEntryDetail
	TotalEntryCount int // Rate内の全エントリ数
}

// AddedEntryDetail: 追加されたエントリの詳細
type AddedEntryDetail struct {
	EntryID    uuid.UUID
	TariffID   uuid.UUID
	TariffName string
}

// ApplyContractToRateError: 契約反映時のエラー詳細
type ApplyContractToRateError struct {
	Code    string // "RATE_NOT_FOUND", "CONTRACT_NOT_CONTRACTED", etc.
	Message string
	Details map[string]any
}

func (e *ApplyContractToRateError) Error() string {
	return e.Message
}

// NewApplyContractToRateError: ApplyContractToRateErrorのファクトリー関数
func NewApplyContractToRateError(code, message string) *ApplyContractToRateError {
	return &ApplyContractToRateError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail: エラーに詳細情報を追加
func (e *ApplyContractToRateError) WithDetail(key string, value any) *ApplyContractToRateError {
	e.Details[key] = value
	return e
}

// UpdateRateEntryTariffInput: レートエントリのTariff差し替えの入力DTO
type UpdateRateEntryTariffInput struct {
	RateID       uuid.UUID // DRAFT状態のレートID
	EntryID      uuid.UUID // 差し替え対象のエントリID
	ContractID   uuid.UUID // 新TariffIDが所属する契約ID
	NewTariffID  uuid.UUID // 新しいTariffID
}

// UpdateRateEntryTariffOutput: レートエントリのTariff差し替えの出力DTO
type UpdateRateEntryTariffOutput struct {
	RateID          uuid.UUID
	RateStatus      string
	EntryID         uuid.UUID
	OldTariffID     uuid.UUID
	NewTariffID     uuid.UUID
	TotalEntryCount int
}

// UpdateRateEntryTariffError: レートエントリTariff差し替え時のエラー詳細
type UpdateRateEntryTariffError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *UpdateRateEntryTariffError) Error() string {
	return e.Message
}

// NewUpdateRateEntryTariffError: UpdateRateEntryTariffErrorのファクトリー関数
func NewUpdateRateEntryTariffError(code, message string) *UpdateRateEntryTariffError {
	return &UpdateRateEntryTariffError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail: エラーに詳細情報を追加
func (e *UpdateRateEntryTariffError) WithDetail(key string, value any) *UpdateRateEntryTariffError {
	e.Details[key] = value
	return e
}
