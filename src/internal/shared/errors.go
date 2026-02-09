package shared

import "fmt"

// DomainErrorCode: ドメインエラーのコード
type DomainErrorCode string

const (
	ErrInvalidArgument       DomainErrorCode = "INVALID_ARGUMENT"
	ErrNotFound              DomainErrorCode = "NOT_FOUND"
	ErrInvalidState          DomainErrorCode = "INVALID_STATE"
	ErrBusinessRuleViolation DomainErrorCode = "BUSINESS_RULE_VIOLATION"
	ErrCurrencyMismatch      DomainErrorCode = "CURRENCY_MISMATCH"
)

// DomainError: 構造化されたドメインエラー
type DomainError struct {
	Code    DomainErrorCode
	Message string
	Details map[string]string
	Cause   error
}

// Error: errorインターフェースの実装
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap: errors.Unwrap対応
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewDomainError: DomainErrorのファクトリ
func NewDomainError(code DomainErrorCode, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Details: make(map[string]string),
	}
}

// WithDetail: 詳細情報を追加するビルダー
func (e *DomainError) WithDetail(key, value string) *DomainError {
	e.Details[key] = value
	return e
}

// WithCause: 原因エラーを追加するビルダー
func (e *DomainError) WithCause(cause error) *DomainError {
	e.Cause = cause
	return e
}

// IsCode: エラーが指定されたコードかどうか判定
func IsCode(err error, code DomainErrorCode) bool {
	if de, ok := err.(*DomainError); ok {
		return de.Code == code
	}
	return false
}
