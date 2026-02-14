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

// DocumentIssued: 書類発行イベント（DRAFT → ISSUED）
type DocumentIssued struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
}

func NewDocumentIssued(documentID, shipmentID uuid.UUID, docType shared.DocType) DocumentIssued {
	return DocumentIssued{
		BaseEvent:  shared.NewBaseEvent("DocumentIssued", documentID, aggregateTypeDocument),
		ShipmentID: shipmentID,
		DocType:    docType,
	}
}

// DocumentReviewSubmitted: レビュー提出イベント
type DocumentReviewSubmitted struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
	ReviewerID uuid.UUID
	Decision   ReviewDecision
}

func NewDocumentReviewSubmitted(documentID, shipmentID uuid.UUID, docType shared.DocType, reviewerID uuid.UUID, decision ReviewDecision) DocumentReviewSubmitted {
	return DocumentReviewSubmitted{
		BaseEvent:  shared.NewBaseEvent("DocumentReviewSubmitted", documentID, aggregateTypeDocument),
		ShipmentID: shipmentID,
		DocType:    docType,
		ReviewerID: reviewerID,
		Decision:   decision,
	}
}

// DocumentRevisionRequested: 修正依頼イベント
type DocumentRevisionRequested struct {
	shared.BaseEvent
	ShipmentID  uuid.UUID
	DocType     shared.DocType
	RequestedBy uuid.UUID
}

func NewDocumentRevisionRequested(documentID, shipmentID uuid.UUID, docType shared.DocType, requestedBy uuid.UUID) DocumentRevisionRequested {
	return DocumentRevisionRequested{
		BaseEvent:   shared.NewBaseEvent("DocumentRevisionRequested", documentID, aggregateTypeDocument),
		ShipmentID:  shipmentID,
		DocType:     docType,
		RequestedBy: requestedBy,
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

// DocumentContentExtracted: 書類コンテンツ抽出完了イベント
type DocumentContentExtracted struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
	DocType    shared.DocType
}

func NewDocumentContentExtracted(documentID, shipmentID uuid.UUID, docType shared.DocType) DocumentContentExtracted {
	return DocumentContentExtracted{
		BaseEvent:  shared.NewBaseEvent("DocumentContentExtracted", documentID, aggregateTypeDocument),
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
