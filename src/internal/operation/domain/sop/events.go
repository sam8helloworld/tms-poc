package sop

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

const aggregateTypeSOPInstance = "SOPInstance"

// SOPInstanceCreated: SOPインスタンス生成イベント
type SOPInstanceCreated struct {
	shared.BaseEvent
	ShipmentID   uuid.UUID
	DefinitionID uuid.UUID
	TaskCount    int
}

func NewSOPInstanceCreated(instanceID, shipmentID, definitionID uuid.UUID, taskCount int) SOPInstanceCreated {
	return SOPInstanceCreated{
		BaseEvent:    shared.NewBaseEvent("SOPInstanceCreated", instanceID, aggregateTypeSOPInstance),
		ShipmentID:   shipmentID,
		DefinitionID: definitionID,
		TaskCount:    taskCount,
	}
}

// SOPTaskStarted: タスク着手イベント
type SOPTaskStarted struct {
	shared.BaseEvent
	TaskID     uuid.UUID
	AssigneeID uuid.UUID
}

func NewSOPTaskStarted(instanceID, taskID, assigneeID uuid.UUID) SOPTaskStarted {
	return SOPTaskStarted{
		BaseEvent:  shared.NewBaseEvent("SOPTaskStarted", instanceID, aggregateTypeSOPInstance),
		TaskID:     taskID,
		AssigneeID: assigneeID,
	}
}

// SOPTaskCompleted: タスク完了イベント
type SOPTaskCompleted struct {
	shared.BaseEvent
	TaskID      uuid.UUID
	CompletedBy uuid.UUID
}

func NewSOPTaskCompleted(instanceID, taskID, completedBy uuid.UUID) SOPTaskCompleted {
	return SOPTaskCompleted{
		BaseEvent:   shared.NewBaseEvent("SOPTaskCompleted", instanceID, aggregateTypeSOPInstance),
		TaskID:      taskID,
		CompletedBy: completedBy,
	}
}

// SOPTaskSkipped: タスクスキップイベント
type SOPTaskSkipped struct {
	shared.BaseEvent
	TaskID uuid.UUID
	Reason string
}

func NewSOPTaskSkipped(instanceID, taskID uuid.UUID, reason string) SOPTaskSkipped {
	return SOPTaskSkipped{
		BaseEvent: shared.NewBaseEvent("SOPTaskSkipped", instanceID, aggregateTypeSOPInstance),
		TaskID:    taskID,
		Reason:    reason,
	}
}

// SOPTaskFailed: タスクエラーイベント
type SOPTaskFailed struct {
	shared.BaseEvent
	TaskID uuid.UUID
	Reason string
}

func NewSOPTaskFailed(instanceID, taskID uuid.UUID, reason string) SOPTaskFailed {
	return SOPTaskFailed{
		BaseEvent: shared.NewBaseEvent("SOPTaskFailed", instanceID, aggregateTypeSOPInstance),
		TaskID:    taskID,
		Reason:    reason,
	}
}

// SOPInstanceCompleted: SOPインスタンス全体完了イベント
type SOPInstanceCompleted struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
}

func NewSOPInstanceCompleted(instanceID, shipmentID uuid.UUID) SOPInstanceCompleted {
	return SOPInstanceCompleted{
		BaseEvent:  shared.NewBaseEvent("SOPInstanceCompleted", instanceID, aggregateTypeSOPInstance),
		ShipmentID: shipmentID,
	}
}

// SOPInstanceCancelled: SOPインスタンスキャンセルイベント
type SOPInstanceCancelled struct {
	shared.BaseEvent
	ShipmentID uuid.UUID
}

func NewSOPInstanceCancelled(instanceID, shipmentID uuid.UUID) SOPInstanceCancelled {
	return SOPInstanceCancelled{
		BaseEvent:  shared.NewBaseEvent("SOPInstanceCancelled", instanceID, aggregateTypeSOPInstance),
		ShipmentID: shipmentID,
	}
}
