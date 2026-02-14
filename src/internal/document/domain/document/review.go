package document

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ==========================================
// DocumentOrigin: 書類の作成元
// ==========================================

// DocumentOrigin: 書類の作成元（誰の責任で作成されたか）
type DocumentOrigin string

const (
	OriginShipper  DocumentOrigin = "SHIPPER"  // 荷主（物流部）が作成
	OriginProvider DocumentOrigin = "PROVIDER" // 物流業者が作成
)

// ==========================================
// DocumentReview: 書類の確認行為
// ==========================================

// ReviewDecision: レビューの判定
type ReviewDecision string

const (
	ReviewApproved          ReviewDecision = "APPROVED"
	ReviewRejected          ReviewDecision = "REJECTED"
	ReviewRevisionRequested ReviewDecision = "REVISION_REQUESTED"
)

// DocumentReview: 書類の確認記録 (Value Object)
// 誰がいつ確認し、どう判断したかを記録する。
type DocumentReview struct {
	ID            uuid.UUID
	ReviewerID    uuid.UUID
	ReviewedAt    time.Time
	Decision      ReviewDecision
	Comment       string
	Discrepancies []Discrepancy
}

// Discrepancy: 確認時に発見した差異
type Discrepancy struct {
	Field    string // 差異があったフィールド
	Expected string // 期待値
	Actual   string // 実際の値
	Severity DiscrepancySeverity
}

// DiscrepancySeverity: 差異の深刻度
type DiscrepancySeverity string

const (
	SeverityInfo    DiscrepancySeverity = "INFO"
	SeverityWarning DiscrepancySeverity = "WARNING"
	SeverityError   DiscrepancySeverity = "ERROR"
)

// NewDocumentReview: DocumentReviewの生成
func NewDocumentReview(
	reviewerID uuid.UUID,
	decision ReviewDecision,
	comment string,
	discrepancies []Discrepancy,
) (*DocumentReview, error) {
	if reviewerID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "reviewer ID is required")
	}
	if decision == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "review decision is required")
	}

	if discrepancies == nil {
		discrepancies = make([]Discrepancy, 0)
	}

	return &DocumentReview{
		ID:            uuid.New(),
		ReviewerID:    reviewerID,
		ReviewedAt:    time.Now(),
		Decision:      decision,
		Comment:       comment,
		Discrepancies: discrepancies,
	}, nil
}

// HasErrors: ERROR深刻度の差異があるか
func (r *DocumentReview) HasErrors() bool {
	for _, d := range r.Discrepancies {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
