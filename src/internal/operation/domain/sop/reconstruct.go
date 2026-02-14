package sop

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ReconstructSOPDefinition: 永続化層からSOPDefinitionを復元するための関数
// ドメインのバリデーションやイベント発行をバイパスしてオブジェクトを再構築する
func ReconstructSOPDefinition(
	id uuid.UUID,
	name, description string,
	targetScope ScopeCriteria,
	steps []SOPStepDefinition,
	status SOPDefinitionStatus,
	version int,
	createdAt, updatedAt time.Time,
) *SOPDefinition {
	return &SOPDefinition{
		ID:          id,
		Name:        name,
		Description: description,
		TargetScope: targetScope,
		steps:       steps,
		status:      status,
		Version:     version,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ReconstructSOPInstance: 永続化層からSOPInstanceを復元するための関数
func ReconstructSOPInstance(
	id, shipmentID, definitionID uuid.UUID,
	definitionName string,
	tasks []SOPTask,
	status SOPInstanceStatus,
	createdAt, updatedAt time.Time,
) *SOPInstance {
	return &SOPInstance{
		ID:             id,
		ShipmentID:     shipmentID,
		DefinitionID:   definitionID,
		DefinitionName: definitionName,
		tasks:          tasks,
		status:         status,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

// ReconstructSOPTask: 永続化層からSOPTaskを復元するための関数
func ReconstructSOPTask(
	id, stepDefinitionID uuid.UUID,
	name, description string,
	orderIndex int,
	actionType ActionType,
	requiredDocTypes []shared.DocType,
	generatedDocType *shared.DocType,
	status TaskStatus,
	assigneeID *uuid.UUID,
	linkedDocumentIDs []uuid.UUID,
	completedAt *time.Time,
	completedBy *uuid.UUID,
	note string,
) SOPTask {
	return SOPTask{
		ID:                id,
		StepDefinitionID:  stepDefinitionID,
		Name:              name,
		Description:       description,
		OrderIndex:        orderIndex,
		ActionType:        actionType,
		RequiredDocTypes:  requiredDocTypes,
		GeneratedDocType:  generatedDocType,
		status:            status,
		AssigneeID:        assigneeID,
		LinkedDocumentIDs: linkedDocumentIDs,
		CompletedAt:       completedAt,
		CompletedBy:       completedBy,
		Note:              note,
	}
}
