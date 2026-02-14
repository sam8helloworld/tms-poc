package document

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ReconstructDocument: 永続化層からDocumentを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructDocument(
	id, shipmentID uuid.UUID,
	docType shared.DocType,
	origin DocumentOrigin,
	fileName, mimeType, storageURI string,
	fileSize int64,
	uploadedBy uuid.UUID,
	status DocumentStatus,
	version int,
	metadata map[string]string,
	content DocumentContent,
	reviews []DocumentReview,
	createdAt, updatedAt time.Time,
) *Document {
	return &Document{
		ID:         id,
		ShipmentID: shipmentID,
		DocType:    docType,
		Origin:     origin,
		FileName:   fileName,
		MimeType:   mimeType,
		StorageURI: storageURI,
		FileSize:   fileSize,
		UploadedBy: uploadedBy,
		status:     status,
		Version:    version,
		Metadata:   metadata,
		content:    content,
		reviews:    reviews,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}
