package contract

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
