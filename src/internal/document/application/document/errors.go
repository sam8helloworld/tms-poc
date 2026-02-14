package document

import "fmt"

// DocumentUseCaseError: Document BCユースケースの共通エラー型
type DocumentUseCaseError struct {
	Code    string
	Message string
	Details map[string]any
}

func NewDocumentUseCaseError(code, message string) *DocumentUseCaseError {
	return &DocumentUseCaseError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *DocumentUseCaseError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DocumentUseCaseError) WithDetail(key string, value any) *DocumentUseCaseError {
	e.Details[key] = value
	return e
}
