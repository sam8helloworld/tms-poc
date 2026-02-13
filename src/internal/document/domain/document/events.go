package document

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

const aggregateTypeDocument = "Document"

// DocumentUploaded: 書類アップロードイベント
type DocumentUploaded struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
	FileName   string
}

func NewDocumentUploaded(documentID, shipmentID uuid.UUID, docType shared.DocType, fileName string) DocumentUploaded {
	return DocumentUploaded{
		BaseEvent:  shared.NewBaseEvent("DocumentUploaded", documentID, aggregateTypeDocument),
		ShipmentID: shipmentID,
		DocType:    docType,
		FileName:   fileName,
	}
}

// DocumentConfirmed: 書類確認完了イベント
type DocumentConfirmed struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
}

func NewDocumentConfirmed(documentID, shipmentID uuid.UUID, docType shared.DocType) DocumentConfirmed {
	return DocumentConfirmed{
		BaseEvent:  shared.NewBaseEvent("DocumentConfirmed", documentID, aggregateTypeDocument),
		ShipmentID: shipmentID,
		DocType:    docType,
	}
}

// DocumentArchived: 書類アーカイブイベント
type DocumentArchived struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
}

func NewDocumentArchived(documentID, shipmentID uuid.UUID, docType shared.DocType) DocumentArchived {
	return DocumentArchived{
		BaseEvent:  shared.NewBaseEvent("DocumentArchived", documentID, aggregateTypeDocument),
		ShipmentID: shipmentID,
		DocType:    docType,
	}
}
