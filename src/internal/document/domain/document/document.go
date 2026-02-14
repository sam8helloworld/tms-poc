package document

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// DocumentStatus: 書類のステータス
type DocumentStatus string

const (
	DocumentStatusDraft             DocumentStatus = "DRAFT"              // 作成中
	DocumentStatusIssued            DocumentStatus = "ISSUED"             // 発行済み（確認待ち）
	DocumentStatusUnderReview       DocumentStatus = "UNDER_REVIEW"       // 確認中
	DocumentStatusRevisionRequested DocumentStatus = "REVISION_REQUESTED" // 修正依頼中
	DocumentStatusConfirmed         DocumentStatus = "CONFIRMED"          // 確認完了
	DocumentStatusArchived          DocumentStatus = "ARCHIVED"           // アーカイブ
)

// Document: 国際物流書類 — 集約ルート
// Shipmentに紐付く各種書類（Invoice、B/L、AWB等）を管理する。
// 書類タイプ別の構造化コンテンツと確認履歴を持つ。
type Document struct {
	shared.EventRecorder

	ID         uuid.UUID
	ShipmentID uuid.UUID // 関連Shipment (cross-BC ID参照)
	DocType    shared.DocType
	Origin     DocumentOrigin // 書類の作成元（SHIPPER or PROVIDER）
	FileName   string
	MimeType   string
	StorageURI string // ストレージのURI/パス
	FileSize   int64  // バイト数
	UploadedBy uuid.UUID
	status     DocumentStatus // private: Status()で取得
	Version    int            // 同一Shipment・同一DocTypeでの版管理
	Metadata   map[string]string // 自由属性（後方互換）

	// 構造化コンテンツ（書類タイプ別のデータ）
	content DocumentContent

	// 確認履歴
	reviews []DocumentReview

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDocument: Documentのファクトリ
func NewDocument(
	shipmentID uuid.UUID,
	docType shared.DocType,
	origin DocumentOrigin,
	fileName string,
	mimeType string,
	storageURI string,
	fileSize int64,
	uploadedBy uuid.UUID,
) (*Document, error) {
	if shipmentID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipment ID is required")
	}
	if docType == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "document type is required")
	}
	if origin == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "document origin is required")
	}
	if fileName == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "file name is required")
	}
	if storageURI == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "storage URI is required")
	}
	if fileSize <= 0 {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "file size must be positive")
	}
	if uploadedBy == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "uploader ID is required")
	}

	now := time.Now()
	doc := &Document{
		ID:         uuid.New(),
		ShipmentID: shipmentID,
		DocType:    docType,
		Origin:     origin,
		FileName:   fileName,
		MimeType:   mimeType,
		StorageURI: storageURI,
		FileSize:   fileSize,
		UploadedBy: uploadedBy,
		status:     DocumentStatusDraft,
		Version:    1,
		Metadata:   make(map[string]string),
		reviews:    make([]DocumentReview, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	doc.RecordEvent(NewDocumentUploaded(doc.ID, shipmentID, docType, fileName))

	return doc, nil
}

// SetContent: 構造化コンテンツを設定し、書類単体のバリデーションを実行
func (d *Document) SetContent(content DocumentContent) error {
	if content == nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "content is required")
	}
	if err := content.Validate(); err != nil {
		return err
	}
	d.content = content
	d.UpdatedAt = time.Now()
	return nil
}

// Content: 構造化コンテンツを返却
func (d *Document) Content() DocumentContent {
	return d.content
}

// Issue: DRAFT → ISSUED（作成者が発行完了、確認待ち状態にする）
func (d *Document) Issue() error {
	if d.status != DocumentStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT documents can be issued")
	}
	d.status = DocumentStatusIssued
	d.UpdatedAt = time.Now()

	d.RecordEvent(NewDocumentIssued(d.ID, d.ShipmentID, d.DocType))

	return nil
}

// StartReview: ISSUED → UNDER_REVIEW
func (d *Document) StartReview() error {
	if d.status != DocumentStatusIssued {
		return shared.NewDomainError(shared.ErrInvalidState, "only ISSUED documents can start review")
	}
	d.status = DocumentStatusUnderReview
	d.UpdatedAt = time.Now()
	return nil
}

// SubmitReview: レビューを提出し、判定に応じてステータスを遷移
func (d *Document) SubmitReview(review DocumentReview) error {
	if d.status != DocumentStatusUnderReview {
		return shared.NewDomainError(shared.ErrInvalidState, "only UNDER_REVIEW documents can receive reviews")
	}

	d.reviews = append(d.reviews, review)
	d.UpdatedAt = time.Now()

	switch review.Decision {
	case ReviewApproved:
		d.status = DocumentStatusConfirmed
		d.RecordEvent(NewDocumentConfirmed(d.ID, d.ShipmentID, d.DocType))
	case ReviewRejected, ReviewRevisionRequested:
		d.status = DocumentStatusRevisionRequested
		d.RecordEvent(NewDocumentRevisionRequested(d.ID, d.ShipmentID, d.DocType, review.ReviewerID))
	}

	d.RecordEvent(NewDocumentReviewSubmitted(d.ID, d.ShipmentID, d.DocType, review.ReviewerID, review.Decision))

	return nil
}

// Revise: REVISION_REQUESTED → DRAFT（修正のために再度DRAFTに戻す）
func (d *Document) Revise() error {
	if d.status != DocumentStatusRevisionRequested {
		return shared.NewDomainError(shared.ErrInvalidState, "only REVISION_REQUESTED documents can be revised")
	}
	d.Version++
	d.status = DocumentStatusDraft
	d.UpdatedAt = time.Now()
	return nil
}

// Confirm: DRAFT → CONFIRMED（レビューなしで直接確認する場合）
func (d *Document) Confirm() error {
	if d.status != DocumentStatusDraft && d.status != DocumentStatusIssued {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT or ISSUED documents can be confirmed directly")
	}
	d.status = DocumentStatusConfirmed
	d.UpdatedAt = time.Now()

	d.RecordEvent(NewDocumentConfirmed(d.ID, d.ShipmentID, d.DocType))

	return nil
}

// Archive: CONFIRMED → ARCHIVED
func (d *Document) Archive() error {
	if d.status != DocumentStatusConfirmed {
		return shared.NewDomainError(shared.ErrInvalidState, "only CONFIRMED documents can be archived")
	}
	d.status = DocumentStatusArchived
	d.UpdatedAt = time.Now()

	d.RecordEvent(NewDocumentArchived(d.ID, d.ShipmentID, d.DocType))

	return nil
}

// UpdateMetadata: メタデータを追加・更新
func (d *Document) UpdateMetadata(key, value string) {
	d.Metadata[key] = value
	d.UpdatedAt = time.Now()
}

// Status: ステータスを返却
func (d *Document) Status() DocumentStatus {
	return d.status
}

// Reviews: 確認履歴のコピーを返却
func (d *Document) Reviews() []DocumentReview {
	result := make([]DocumentReview, len(d.reviews))
	copy(result, d.reviews)
	return result
}

// LatestReview: 最新のレビューを返却
func (d *Document) LatestReview() *DocumentReview {
	if len(d.reviews) == 0 {
		return nil
	}
	review := d.reviews[len(d.reviews)-1]
	return &review
}
