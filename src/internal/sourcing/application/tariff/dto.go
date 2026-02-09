package tariff

import (
	"io"
	"time"

	"github.com/google/uuid"
)

// RegisterTariffInput: 料金表登録ユースケースの入力DTO
// 入札プロセスにおいて、物流企業から提示された料金表を登録する
type RegisterTariffInput struct {
	// ファイル情報
	FileReader io.Reader // アップロードされたファイルのデータストリーム
	FileFormat string    // "pdf", "xlsx", "csv"
	FileName   string    // "ACME_2026_Export_Rates.pdf"
	FileSize   int64
	UploadedBy uuid.UUID // アップロードしたユーザーのID
	UploadedAt time.Time

	// 契約情報（DRAFT契約の識別または新規作成）
	ContractID *uuid.UUID // 既存のDRAFT契約に追加する場合は指定
	ProviderID uuid.UUID  // 物流企業ID（新規契約作成時に必須）
	ShipperID  uuid.UUID  // 荷主企業ID（新規契約作成時に必須）

	// 契約有効期間（新規契約作成時に使用）
	ContractValidFrom *time.Time
	ContractValidTo   *time.Time

	// オプション: ファイルから取得できない場合はマニュアルで指定
	OverrideTariffName    *string    // ファイルから名前を取得できない場合に使用
	OverrideEffectiveFrom *time.Time // ファイルから有効期間を取得できない場合に使用
	OverrideEffectiveTo   *time.Time
}

// RegisterTariffOutput: 料金表登録ユースケースの出力DTO
type RegisterTariffOutput struct {
	ContractID       uuid.UUID
	ContractStatus   string // "DRAFT", "CONTRACTED"
	TariffID         uuid.UUID
	TariffName       string
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	LineItemCount    int
	CreatedAt        time.Time
	IsNewContract    bool // 新規に契約を作成した場合はtrue
	IsUpdatedTariff  bool // 既存のTariffを更新した場合はtrue
	TotalTariffCount int  // この契約に含まれる料金表の総数
}

// RegisterTariffError: 料金表登録時のエラー詳細
type RegisterTariffError struct {
	Code    string // "FILE_PARSE_ERROR", "VALIDATION_ERROR", "CONTRACT_NOT_FOUND"
	Message string
	Details map[string]any
	Cause   error // 原因となったエラー
}

func (e *RegisterTariffError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap: 原因エラーを取得
func (e *RegisterTariffError) Unwrap() error {
	return e.Cause
}

// NewRegisterTariffError: RegisterTariffErrorのファクトリー関数
func NewRegisterTariffError(code, message string) *RegisterTariffError {
	return &RegisterTariffError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail: エラーに詳細情報を追加
func (e *RegisterTariffError) WithDetail(key string, value any) *RegisterTariffError {
	e.Details[key] = value
	return e
}

// WithCause: 原因エラーを追加
func (e *RegisterTariffError) WithCause(cause error) *RegisterTariffError {
	e.Cause = cause
	return e
}

// AmendContractTariffInput: 契約アメンドメント（料金表改定）の入力DTO
type AmendContractTariffInput struct {
	FileReader    io.Reader
	FileFormat    string    // "csv", "excel", "json"
	FileName      string
	ContractID    uuid.UUID // CONTRACTED状態の契約ID
	BaseTariffID  uuid.UUID // 改定元のTariffID
	UploadedBy    uuid.UUID
	EffectiveFrom *time.Time // 新バージョンの有効期間開始（オプション）
	EffectiveTo   *time.Time // 新バージョンの有効期間終了（オプション）
}

// AmendContractTariffOutput: 契約アメンドメントの出力DTO
type AmendContractTariffOutput struct {
	ContractID       uuid.UUID
	ContractStatus   string
	TariffID         uuid.UUID
	TariffName       string
	TariffVersion    int
	BaseTariffID     uuid.UUID
	EffectiveFrom    time.Time
	EffectiveTo      time.Time
	LineItemCount    int
	TotalTariffCount int
	Message          string
}

// AmendTariffError: 契約アメンドメント時のエラー詳細
type AmendTariffError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *AmendTariffError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap: 原因エラーを取得
func (e *AmendTariffError) Unwrap() error {
	return e.Cause
}

// NewAmendTariffError: AmendTariffErrorのファクトリー関数
func NewAmendTariffError(code, message string) *AmendTariffError {
	return &AmendTariffError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail: エラーに詳細情報を追加
func (e *AmendTariffError) WithDetail(key string, value any) *AmendTariffError {
	e.Details[key] = value
	return e
}

// WithCause: 原因エラーを追加
func (e *AmendTariffError) WithCause(cause error) *AmendTariffError {
	e.Cause = cause
	return e
}
