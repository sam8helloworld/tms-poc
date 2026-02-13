package sop

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// SOPDefinition: SOP（Standard Operating Procedure）テンプレート — 集約ルート
// 国際物流業務の標準手順を定義する。DRAFT状態でステップを編集し、ACTIVEにして運用開始する。
type SOPDefinition struct {
	shared.EventRecorder

	ID          uuid.UUID
	Name        string // 例: "海上輸出標準フロー（US向け）"
	Description string
	TargetScope ScopeCriteria
	steps       []SOPStepDefinition // private: Steps()で取得
	status      SOPDefinitionStatus // private: Status()で取得
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SOPStepDefinition: SOPの各ステップ定義（Entity）
type SOPStepDefinition struct {
	ID               uuid.UUID
	Name             string // 例: "Arrival Notice送付"
	Description      string
	OrderIndex       int
	RequiredDocTypes []shared.DocType  // 完了に必要な書類種別
	GeneratedDocType *shared.DocType   // このステップで生成される書類種別
	ActionType       ActionType
	IsAutomatable    bool // Phase 2で自動化可能かのメタデータ
}

// NewSOPDefinition: SOPDefinitionのファクトリ
func NewSOPDefinition(name, description string, scope ScopeCriteria) (*SOPDefinition, error) {
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "SOP definition name is required")
	}

	now := time.Now()
	return &SOPDefinition{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		TargetScope: scope,
		steps:       make([]SOPStepDefinition, 0),
		status:      DefinitionStatusDraft,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// AddStep: ステップを追加（DRAFT状態のみ）
func (d *SOPDefinition) AddStep(step SOPStepDefinition) error {
	if d.status != DefinitionStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "steps can only be added in DRAFT status")
	}
	if step.Name == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "step name is required")
	}
	if step.ID == uuid.Nil {
		step.ID = uuid.New()
	}
	d.steps = append(d.steps, step)
	d.UpdatedAt = time.Now()
	return nil
}

// RemoveStep: ステップを削除（DRAFT状態のみ）
func (d *SOPDefinition) RemoveStep(stepID uuid.UUID) error {
	if d.status != DefinitionStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "steps can only be removed in DRAFT status")
	}
	for i, s := range d.steps {
		if s.ID == stepID {
			d.steps = append(d.steps[:i], d.steps[i+1:]...)
			d.UpdatedAt = time.Now()
			return nil
		}
	}
	return shared.NewDomainError(shared.ErrNotFound, "step not found")
}

// ReorderSteps: ステップの順序を変更
func (d *SOPDefinition) ReorderSteps(stepIDs []uuid.UUID) error {
	if d.status != DefinitionStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "steps can only be reordered in DRAFT status")
	}
	if len(stepIDs) != len(d.steps) {
		return shared.NewDomainError(shared.ErrInvalidArgument, "stepIDs count must match current steps count")
	}

	stepMap := make(map[uuid.UUID]SOPStepDefinition, len(d.steps))
	for _, s := range d.steps {
		stepMap[s.ID] = s
	}

	reordered := make([]SOPStepDefinition, 0, len(stepIDs))
	for i, id := range stepIDs {
		step, ok := stepMap[id]
		if !ok {
			return shared.NewDomainError(shared.ErrNotFound, "step not found: "+id.String())
		}
		step.OrderIndex = i
		reordered = append(reordered, step)
	}

	d.steps = reordered
	d.UpdatedAt = time.Now()
	return nil
}

// Activate: DRAFT → ACTIVE（1つ以上のステップ必須）
func (d *SOPDefinition) Activate() error {
	if d.status != DefinitionStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT definitions can be activated")
	}
	if len(d.steps) == 0 {
		return shared.NewDomainError(shared.ErrBusinessRuleViolation, "at least one step is required to activate")
	}
	d.status = DefinitionStatusActive
	d.UpdatedAt = time.Now()
	return nil
}

// Archive: ACTIVE → ARCHIVED
func (d *SOPDefinition) Archive() error {
	if d.status != DefinitionStatusActive {
		return shared.NewDomainError(shared.ErrInvalidState, "only ACTIVE definitions can be archived")
	}
	d.status = DefinitionStatusArchived
	d.UpdatedAt = time.Now()
	return nil
}

// Steps: ステップのコピーを返却
func (d *SOPDefinition) Steps() []SOPStepDefinition {
	result := make([]SOPStepDefinition, len(d.steps))
	copy(result, d.steps)
	return result
}

// Status: ステータスを返却
func (d *SOPDefinition) Status() SOPDefinitionStatus {
	return d.status
}
