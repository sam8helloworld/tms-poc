package sop

import "github.com/sam8helloworld/tms-poc/internal/shared"

// ActionType: SOPステップで実行するアクションの種別
type ActionType string

const (
	ActionDocumentUpload    ActionType = "DOCUMENT_UPLOAD"    // 書類アップロード
	ActionDocumentGeneration ActionType = "DOCUMENT_GENERATION" // 書類自動生成
	ActionEmailSend         ActionType = "EMAIL_SEND"         // メール送信
	ActionApprovalRequest   ActionType = "APPROVAL_REQUEST"   // 承認依頼
	ActionExternalAPICall   ActionType = "EXTERNAL_API_CALL"  // 外部API呼び出し
	ActionManualCheck       ActionType = "MANUAL_CHECK"       // 手動確認
)

// TaskStatus: SOPタスクの実行ステータス
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusSkipped    TaskStatus = "SKIPPED"
	TaskStatusError      TaskStatus = "ERROR"
)

// SOPInstanceStatus: SOPインスタンス全体のステータス
type SOPInstanceStatus string

const (
	InstanceStatusInProgress SOPInstanceStatus = "IN_PROGRESS"
	InstanceStatusCompleted  SOPInstanceStatus = "COMPLETED"
	InstanceStatusCancelled  SOPInstanceStatus = "CANCELLED"
)

// SOPDefinitionStatus: SOPテンプレートのステータス
type SOPDefinitionStatus string

const (
	DefinitionStatusDraft    SOPDefinitionStatus = "DRAFT"
	DefinitionStatusActive   SOPDefinitionStatus = "ACTIVE"
	DefinitionStatusArchived SOPDefinitionStatus = "ARCHIVED"
)

// ScopeCriteria: SOPの適用条件（Value Object）
type ScopeCriteria struct {
	Direction          shared.TradeDirection // EXPORT / IMPORT
	TransportMode      shared.TransportMode // OCEAN / AIR / TRUCK
	OriginCountryCode  *string              // ISO 3166-1 alpha-2 (optional)
	DestCountryCode    *string              // ISO 3166-1 alpha-2 (optional)
}

// Matches: 指定された条件がこのScopeに合致するかを判定
func (s ScopeCriteria) Matches(direction shared.TradeDirection, mode shared.TransportMode, originCountry, destCountry *string) bool {
	if s.Direction != direction {
		return false
	}
	if s.TransportMode != mode {
		return false
	}
	if s.OriginCountryCode != nil {
		if originCountry == nil || *s.OriginCountryCode != *originCountry {
			return false
		}
	}
	if s.DestCountryCode != nil {
		if destCountry == nil || *s.DestCountryCode != *destCountry {
			return false
		}
	}
	return true
}
