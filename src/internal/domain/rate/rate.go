package rate

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// RateStatus: レートのステータス
type RateStatus string

const (
	// RateStatusDraft: ドラフト（作成中）
	RateStatusDraft RateStatus = "DRAFT"

	// RateStatusActive: 有効（使用可能）
	RateStatusActive RateStatus = "ACTIVE"

	// RateStatusExpired: 期限切れ
	RateStatusExpired RateStatus = "EXPIRED"
)

// Rate: 社内レート (Aggregate Root)
// 荷主が複数業者のTariffからルート単位で選択・組み合わせた通期レート
type Rate struct {
	shared.EventRecorder

	ID          uuid.UUID
	ShipperID   uuid.UUID
	Name        string // e.g. "2026年上期 固定レート"
	ValidPeriod shared.DateRange
	status      RateStatus
	entries     []*RateEntry
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewRate: Rateのファクトリ関数（DRAFT状態で生成）
func NewRate(
	shipperID uuid.UUID,
	name string,
	validFrom time.Time,
	validTo time.Time,
) (*Rate, error) {
	if shipperID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "shipperID is required")
	}
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "rate name is required")
	}
	if validFrom.After(validTo) {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "valid period is invalid: from must be before or equal to to")
	}

	now := time.Now()
	return &Rate{
		ID:        uuid.New(),
		ShipperID: shipperID,
		Name:      name,
		ValidPeriod: shared.DateRange{
			From: validFrom,
			To:   validTo,
		},
		status:    RateStatusDraft,
		entries:   make([]*RateEntry, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Status: ステータスのgetter
func (r *Rate) Status() RateStatus {
	return r.status
}

// Entries: エントリのgetter（コピーを返却）
func (r *Rate) Entries() []*RateEntry {
	result := make([]*RateEntry, len(r.entries))
	copy(result, r.entries)
	return result
}

// AddEntry: レートエントリを追加（DRAFT状態でのみ可）
func (r *Rate) AddEntry(entry *RateEntry) error {
	if r.status != RateStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "entries can only be added in DRAFT status")
	}
	if entry == nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "entry is required")
	}
	if entry.TariffID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "tariffID is required")
	}

	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	r.entries = append(r.entries, entry)
	r.UpdatedAt = time.Now()
	r.RecordEvent(NewRateEntryAdded(r.ID, entry.ID))
	return nil
}

// RemoveEntry: レートエントリを削除（DRAFT状態でのみ可）
func (r *Rate) RemoveEntry(entryID uuid.UUID) error {
	if r.status != RateStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "entries can only be removed in DRAFT status")
	}

	for i, entry := range r.entries {
		if entry.ID == entryID {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			r.UpdatedAt = time.Now()
			return nil
		}
	}

	return shared.NewDomainError(shared.ErrNotFound, "entry not found")
}

// Activate: DRAFT → ACTIVE（エントリが1つ以上必要）
func (r *Rate) Activate() error {
	if r.status != RateStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "only DRAFT rates can be activated")
	}
	if len(r.entries) == 0 {
		return shared.NewDomainError(shared.ErrBusinessRuleViolation, "rate must have at least one entry to be activated")
	}

	r.status = RateStatusActive
	r.UpdatedAt = time.Now()
	r.RecordEvent(NewRateActivated(r.ID))
	return nil
}

// MarkAsExpired: ACTIVE → EXPIRED
func (r *Rate) MarkAsExpired() error {
	if r.status != RateStatusActive {
		return shared.NewDomainError(shared.ErrInvalidState, "only ACTIVE rates can be marked as expired")
	}

	r.status = RateStatusExpired
	r.UpdatedAt = time.Now()
	return nil
}

// ReplaceEntryTariff: エントリのTariffIDを新しいTariffIDに差し替え（DRAFT状態でのみ可）
func (r *Rate) ReplaceEntryTariff(entryID uuid.UUID, newTariffID uuid.UUID) error {
	if r.status != RateStatusDraft {
		return shared.NewDomainError(shared.ErrInvalidState, "entry tariffs can only be replaced in DRAFT status")
	}
	if newTariffID == uuid.Nil {
		return shared.NewDomainError(shared.ErrInvalidArgument, "new tariffID is required")
	}

	for _, entry := range r.entries {
		if entry.ID == entryID {
			oldTariffID := entry.TariffID
			entry.TariffID = newTariffID
			r.UpdatedAt = time.Now()
			r.RecordEvent(NewRateEntryTariffReplaced(r.ID, entryID, oldTariffID, newTariffID))
			return nil
		}
	}

	return shared.NewDomainError(shared.ErrNotFound, "entry not found")
}

// FindEntriesForRoute: ルートに一致するエントリを検索
func (r *Rate) FindEntriesForRoute(originID, destID route.LocationID, mode shared.TransportMode) []*RateEntry {
	var matched []*RateEntry
	for _, entry := range r.entries {
		if entry.RouteScope.Matches(originID, destID, mode) {
			matched = append(matched, entry)
		}
	}
	return matched
}

// RateEntry: レートの構成要素
// あるルート範囲に対して、特定の業者の特定のTariffをまるごと採用する
type RateEntry struct {
	ID         uuid.UUID
	ProviderID uuid.UUID        // 業者
	ContractID uuid.UUID        // 契約
	TariffID   uuid.UUID        // Tariffまるごと採用
	RouteScope RouteScope       // 適用ルート範囲
}

// RouteScope: レートエントリの適用ルート範囲
type RouteScope struct {
	OriginID      *route.LocationID    // nil = 全Origin
	DestinationID *route.LocationID    // nil = 全Destination
	TransportMode *shared.TransportMode // nil = 全モード
}

// Matches: 指定されたルートがこのスコープに一致するか判定
func (rs RouteScope) Matches(originID, destID route.LocationID, mode shared.TransportMode) bool {
	if rs.OriginID != nil && *rs.OriginID != originID {
		return false
	}
	if rs.DestinationID != nil && *rs.DestinationID != destID {
		return false
	}
	if rs.TransportMode != nil && *rs.TransportMode != mode {
		return false
	}
	return true
}
