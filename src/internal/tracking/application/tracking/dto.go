package tracking

import (
	"time"

	"github.com/google/uuid"
	domain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ==========================================
// RegisterShipmentTracking DTO
// ==========================================

// SegmentInput: セグメント入力VO
type SegmentInput struct {
	ActualOriginLocationID uuid.UUID
	ActualDestLocationID   uuid.UUID
	Mode                   shared.TransportMode
	CarrierTrackingNumber  string
	PrimarySource          domain.TrackingSourceType
}

// RegisterShipmentTrackingInput: トラッキング登録ユースケースの入力DTO
type RegisterShipmentTrackingInput struct {
	ShipmentID         uuid.UUID
	TrackingNumber     string
	TrackingNumberType domain.TrackingNumberType
	CarrierID          uuid.UUID
	Segments           []SegmentInput
}

// RegisterShipmentTrackingOutput: トラッキング登録ユースケースの出力DTO
type RegisterShipmentTrackingOutput struct {
	TrackingUnitID uuid.UUID
	ShipmentID     uuid.UUID
	TrackingNumber string
	SegmentCount   int
	Status         shared.TrackingStatus
	CreatedAt      time.Time
}

// RegisterTrackingError: トラッキング登録時のエラー詳細
type RegisterTrackingError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *RegisterTrackingError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *RegisterTrackingError) Unwrap() error {
	return e.Cause
}

// NewRegisterTrackingError: RegisterTrackingErrorのファクトリー関数
func NewRegisterTrackingError(code, message string) *RegisterTrackingError {
	return &RegisterTrackingError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *RegisterTrackingError) WithDetail(key string, value any) *RegisterTrackingError {
	e.Details[key] = value
	return e
}

func (e *RegisterTrackingError) WithCause(cause error) *RegisterTrackingError {
	e.Cause = cause
	return e
}

// ==========================================
// SyncTrackingEvents DTO
// ==========================================

// SyncTrackingInput: トラッキングイベント同期ユースケースの入力DTO
type SyncTrackingInput struct {
	TrackingUnitID uuid.UUID
}

// SyncedSegmentDetail: セグメント単位の同期結果詳細
type SyncedSegmentDetail struct {
	SegmentID      uuid.UUID
	NewEventsCount int
	LatestStatus   shared.TrackingStatus
	Error          string // セグメント単位のエラー（部分的成功を許容）
}

// SyncTrackingOutput: トラッキングイベント同期ユースケースの出力DTO
type SyncTrackingOutput struct {
	TrackingUnitID uuid.UUID
	OverallStatus  shared.TrackingStatus
	SyncedSegments []SyncedSegmentDetail
	TotalNewEvents int
	SyncedAt       time.Time
}

// SyncTrackingError: トラッキングイベント同期時のエラー詳細
type SyncTrackingError struct {
	Code    string
	Message string
	Details map[string]any
	Cause   error
}

func (e *SyncTrackingError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *SyncTrackingError) Unwrap() error {
	return e.Cause
}

// NewSyncTrackingError: SyncTrackingErrorのファクトリー関数
func NewSyncTrackingError(code, message string) *SyncTrackingError {
	return &SyncTrackingError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *SyncTrackingError) WithDetail(key string, value any) *SyncTrackingError {
	e.Details[key] = value
	return e
}

func (e *SyncTrackingError) WithCause(cause error) *SyncTrackingError {
	e.Cause = cause
	return e
}
