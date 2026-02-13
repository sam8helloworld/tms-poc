package document

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// DocumentStatus: 書類のステータス
type DocumentStatus string

const (
	DocumentStatusDraft     DocumentStatus = "DRAFT"
	DocumentStatusConfirmed DocumentStatus = "CONFIRMED"
	DocumentStatusArchived  DocumentStatus = "ARCHIVED"
)

// Document: 国際物流書類 — 集約ルート
// Shipmentに紐付く各種書類（Invoice、B/L、AWB等）を管理する。
type Document struct {
	shared.EventRecorder

	ID         uuid.UUID
	ShipmentID uuid.UUID // 関連Shipment (cross-BC ID参照)
	DocType    shared.DocType
	FileName   string
	MimeType   string
	StorageURI string // ストレージのURI/パス
	FileSize   int64  // バイト数
	UploadedBy uuid.UUID
	status     DocumentStatus // private: Status()で取得
	Version    int            // 同一Shipment・同一DocTypeでの版管理
	Metadata   map[string]string // 自由属性（B/L番号、Invoice番号等）
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewDocument: Documentのファクトリ
func NewDocument(
	shipmentID uuid.UUID,
	docType shared.DocType,
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
		FileName:   fileName,
		MimeType:   mimeType,
		StorageURI: storageURI,
		FileSize:   fileSize,
		UploadedBy: uploadedBy,
		status:     DocumentStatusDraft,
		Version:    1,
		Metadata:   make(map[string]string),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	doc.RecordEvent(NewDocumentUploaded(doc.ID, shipmentID, docType, fileName))

	return doc, nil
}

// Confirm: DRAFT → CONFIRMED
func (d *Document) Confirm() error {
	if d.status != DocumentStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT documents can be confirmed")
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
