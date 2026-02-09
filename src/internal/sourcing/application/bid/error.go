package bid

// CreateBidContractError: 入札契約作成時のエラー詳細
type CreateBidContractError struct {
	Code    string // "INVALID_INPUT", "DUPLICATE_CONTRACT", "PROVIDER_NOT_FOUND"
	Message string
	Details map[string]any
}

func (e *CreateBidContractError) Error() string {
	return e.Message
}

// NewCreateBidContractError: CreateBidContractErrorのファクトリー関数
func NewCreateBidContractError(code, message string) *CreateBidContractError {
	return &CreateBidContractError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// WithDetail: エラーに詳細情報を追加
func (e *CreateBidContractError) WithDetail(key string, value any) *CreateBidContractError {
	e.Details[key] = value
	return e
}
