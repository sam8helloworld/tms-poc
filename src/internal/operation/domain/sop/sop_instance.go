package sop

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// SOPInstance: Shipmentに紐付くSOPの実行インスタンス — 集約ルート
// SOPDefinitionから生成され、各タスクの進捗を管理する。
type SOPInstance struct {
	shared.EventRecorder

	ID             uuid.UUID
	ShipmentID     uuid.UUID // 対象Shipment (cross-BC ID参照)
	DefinitionID   uuid.UUID // 元のSOPDefinition (ID参照)
	DefinitionName string    // コピー（定義が変わっても記録を残す）
	tasks          []SOPTask // private: Tasks()で取得
	status         SOPInstanceStatus // private: Status()で取得
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SOPTask: SOPインスタンス内の個別タスク（Entity）
type SOPTask struct {
	ID               uuid.UUID
	StepDefinitionID uuid.UUID // 元のSOPStepDefinitionのID
	Name             string
	Description      string
	OrderIndex       int
	ActionType       ActionType
	RequiredDocTypes []shared.DocType
	GeneratedDocType *shared.DocType
	status           TaskStatus // private: Status()で取得
	AssigneeID       *uuid.UUID
	LinkedDocumentIDs []uuid.UUID // Document BCへのID参照
	CompletedAt      *time.Time
	CompletedBy      *uuid.UUID
	Note             string
}

// Status: タスクのステータスを返却
func (t SOPTask) Status() TaskStatus {
	return t.status
}

// NewSOPInstance: SOPDefinitionからSOPInstanceを生成（Hydrate）
func NewSOPInstance(shipmentID uuid.UUID, definition *SOPDefinition) (*SOPInstance, error) {
	if shipmentID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipment ID is required")
	}
	if definition == nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "SOP definition is required")
	}
	if definition.Status() != DefinitionStatusActive {
		return nil, shared.NewDomainError(shared.ErrInvalidState, "SOP definition must be ACTIVE to create an instance")
	}

	now := time.Now()
	instanceID := uuid.New()

	steps := definition.Steps()
	tasks := make([]SOPTask, len(steps))
	for i, step := range steps {
		reqDocTypes := make([]shared.DocType, len(step.RequiredDocTypes))
		copy(reqDocTypes, step.RequiredDocTypes)

		tasks[i] = SOPTask{
			ID:               uuid.New(),
			StepDefinitionID: step.ID,
			Name:             step.Name,
			Description:      step.Description,
			OrderIndex:       step.OrderIndex,
			ActionType:       step.ActionType,
			RequiredDocTypes: reqDocTypes,
			GeneratedDocType: step.GeneratedDocType,
			status:           TaskStatusPending,
			LinkedDocumentIDs: make([]uuid.UUID, 0),
		}
	}

	instance := &SOPInstance{
		ID:             instanceID,
		ShipmentID:     shipmentID,
		DefinitionID:   definition.ID,
		DefinitionName: definition.Name,
		tasks:          tasks,
		status:         InstanceStatusInProgress,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	instance.RecordEvent(NewSOPInstanceCreated(instanceID, shipmentID, definition.ID, len(tasks)))

	return instance, nil
}

// StartTask: タスクを着手する（PENDING → IN_PROGRESS）
func (inst *SOPInstance) StartTask(taskID uuid.UUID, assigneeID uuid.UUID) error {
	if err := inst.checkNotCancelled(); err != nil {
		return err
	}

	task, err := inst.findMutableTask(taskID)
	if err != nil {
		return err
	}

	if task.status != TaskStatusPending {
		return shared.NewDomainError(shared.ErrInvalidState, "task can only be started from PENDING status")
	}

	task.status = TaskStatusInProgress
	task.AssigneeID = &assigneeID
	inst.UpdatedAt = time.Now()

	inst.RecordEvent(NewSOPTaskStarted(inst.ID, taskID, assigneeID))

	return nil
}

// CompleteTask: タスクを完了する（IN_PROGRESS → COMPLETED）
// 全タスク完了時にInstance自体もCOMPLETEDに自動遷移する。
func (inst *SOPInstance) CompleteTask(taskID uuid.UUID, completedBy uuid.UUID, note string) error {
	if err := inst.checkNotCancelled(); err != nil {
		return err
	}

	task, err := inst.findMutableTask(taskID)
	if err != nil {
		return err
	}

	if task.status != TaskStatusInProgress {
		return shared.NewDomainError(shared.ErrInvalidState, "task can only be completed from IN_PROGRESS status")
	}

	now := time.Now()
	task.status = TaskStatusCompleted
	task.CompletedAt = &now
	task.CompletedBy = &completedBy
	task.Note = note
	inst.UpdatedAt = now

	inst.RecordEvent(NewSOPTaskCompleted(inst.ID, taskID, completedBy))

	inst.checkAllTasksDone()

	return nil
}

// SkipTask: タスクをスキップする（→ SKIPPED）
func (inst *SOPInstance) SkipTask(taskID uuid.UUID, reason string) error {
	if err := inst.checkNotCancelled(); err != nil {
		return err
	}

	task, err := inst.findMutableTask(taskID)
	if err != nil {
		return err
	}

	if task.status == TaskStatusCompleted || task.status == TaskStatusSkipped {
		return shared.NewDomainError(shared.ErrInvalidState, "completed or skipped tasks cannot be modified")
	}

	task.status = TaskStatusSkipped
	task.Note = reason
	inst.UpdatedAt = time.Now()

	inst.RecordEvent(NewSOPTaskSkipped(inst.ID, taskID, reason))

	inst.checkAllTasksDone()

	return nil
}

// FailTask: タスクをエラー状態にする（→ ERROR）
func (inst *SOPInstance) FailTask(taskID uuid.UUID, reason string) error {
	if err := inst.checkNotCancelled(); err != nil {
		return err
	}

	task, err := inst.findMutableTask(taskID)
	if err != nil {
		return err
	}

	if task.status == TaskStatusCompleted || task.status == TaskStatusSkipped {
		return shared.NewDomainError(shared.ErrInvalidState, "completed or skipped tasks cannot be modified")
	}

	task.status = TaskStatusError
	task.Note = reason
	inst.UpdatedAt = time.Now()

	inst.RecordEvent(NewSOPTaskFailed(inst.ID, taskID, reason))

	return nil
}

// LinkDocumentToTask: タスクに書類を紐付ける
func (inst *SOPInstance) LinkDocumentToTask(taskID uuid.UUID, documentID uuid.UUID) error {
	if err := inst.checkNotCancelled(); err != nil {
		return err
	}

	task, err := inst.findMutableTask(taskID)
	if err != nil {
		return err
	}

	// 重複チェック
	for _, id := range task.LinkedDocumentIDs {
		if id == documentID {
			return shared.NewDomainError(shared.ErrBusinessRuleViolation, "document already linked to this task")
		}
	}

	task.LinkedDocumentIDs = append(task.LinkedDocumentIDs, documentID)
	inst.UpdatedAt = time.Now()
	return nil
}

// Cancel: Instance全体をキャンセル
func (inst *SOPInstance) Cancel() error {
	if inst.status == InstanceStatusCancelled {
		return shared.NewDomainError(shared.ErrInvalidState, "instance is already cancelled")
	}
	if inst.status == InstanceStatusCompleted {
		return shared.NewDomainError(shared.ErrInvalidState, "completed instance cannot be cancelled")
	}

	inst.status = InstanceStatusCancelled
	inst.UpdatedAt = time.Now()

	inst.RecordEvent(NewSOPInstanceCancelled(inst.ID, inst.ShipmentID))

	return nil
}

// Tasks: タスクのコピーを返却
func (inst *SOPInstance) Tasks() []SOPTask {
	result := make([]SOPTask, len(inst.tasks))
	copy(result, inst.tasks)
	return result
}

// Status: ステータスを返却
func (inst *SOPInstance) Status() SOPInstanceStatus {
	return inst.status
}

// FindTaskByID: IDでタスクを検索（コピーを返却）
func (inst *SOPInstance) FindTaskByID(id uuid.UUID) *SOPTask {
	for _, t := range inst.tasks {
		if t.ID == id {
			task := t
			return &task
		}
	}
	return nil
}

// Progress: 進捗率（完了+スキップ数, 全体数）
func (inst *SOPInstance) Progress() (completed int, total int) {
	total = len(inst.tasks)
	for _, t := range inst.tasks {
		if t.status == TaskStatusCompleted || t.status == TaskStatusSkipped {
			completed++
		}
	}
	return completed, total
}

// checkNotCancelled: キャンセル済みでないことを確認
func (inst *SOPInstance) checkNotCancelled() error {
	if inst.status == InstanceStatusCancelled {
		return shared.NewDomainError(shared.ErrInvalidState, "cannot modify a cancelled instance")
	}
	return nil
}

// findMutableTask: IDでタスクへのポインタを取得（内部操作用）
func (inst *SOPInstance) findMutableTask(taskID uuid.UUID) (*SOPTask, error) {
	for i := range inst.tasks {
		if inst.tasks[i].ID == taskID {
			return &inst.tasks[i], nil
		}
	}
	return nil, shared.NewDomainError(shared.ErrNotFound, "task not found")
}

// checkAllTasksDone: 全タスクがCOMPLETED or SKIPPEDならInstanceをCOMPLETEDにする
func (inst *SOPInstance) checkAllTasksDone() {
	for _, t := range inst.tasks {
		if t.status != TaskStatusCompleted && t.status != TaskStatusSkipped {
			return
		}
	}
	inst.status = InstanceStatusCompleted
	inst.RecordEvent(NewSOPInstanceCompleted(inst.ID, inst.ShipmentID))
}
