package shipment

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// ==========================================
// ShipmentExecution: 出荷実行コンテキスト
// ==========================================

// MilestoneType: マイルストーンの種別
// 各社の物流部毎にマイルストーンは異なるため、stringベースで拡張可能にする
type MilestoneType string

// 標準的なマイルストーン種別（よく使われるもの）
const (
	MilestoneBookingConfirmed    MilestoneType = "BOOKING_CONFIRMED"
	MilestoneShippingInstruction MilestoneType = "SHIPPING_INSTRUCTION_ISSUED"
	MilestoneShipped             MilestoneType = "SHIPPED"
	MilestoneCustomsExportFiled  MilestoneType = "CUSTOMS_EXPORT_FILED"
	MilestoneCustomsExportCleared MilestoneType = "CUSTOMS_EXPORT_CLEARED"
	MilestoneCustomsImportFiled  MilestoneType = "CUSTOMS_IMPORT_FILED"
	MilestoneCustomsImportCleared MilestoneType = "CUSTOMS_IMPORT_CLEARED"
	MilestoneArrived             MilestoneType = "ARRIVED"
	MilestoneDelivered           MilestoneType = "DELIVERED"
	MilestoneInvoiceReceived     MilestoneType = "INVOICE_RECEIVED"
)

// Milestone: 出荷マイルストーン (Value Object)
// 書類の確認を通じて確定した業務的事実を記録する。
// 同一MilestoneTypeで複数記録可能（履歴保持）。
type Milestone struct {
	ID               uuid.UUID        // マイルストーンID
	Type             MilestoneType    // マイルストーン種別
	OccurredAt       time.Time        // この事実がいつ発生したか
	RecordedAt       time.Time        // いつシステムに記録されたか
	SourceDocumentID uuid.UUID        // どの書類由来か
	SourceDocType    shared.DocType   // 書類種別
	Payload          MilestonePayload // マイルストーン別の確定事実
	Sequence         int              // 同一Type内の順序（1始まり）
}

// MilestonePayload: マイルストーン別の確定事実を表すinterface
type MilestonePayload interface {
	milestoneType() MilestoneType
}

// ==========================================
// 標準的な MilestonePayload 実装
// ==========================================

// BookingConfirmedPayload: Booking Confirmation から確定する事実
type BookingConfirmedPayload struct {
	BookingNo  string
	VesselName string
	VoyageNo   string
	ETD        time.Time
	ETA        time.Time
}

func (p BookingConfirmedPayload) milestoneType() MilestoneType {
	return MilestoneBookingConfirmed
}

// ShippedPayload: B/L or AWB から確定する事実
type ShippedPayload struct {
	TransportDocNo string    // B/L番号 or AWB番号
	OnBoardDate    time.Time // 船積日 or 搭載日
	VesselName     string    // 船名 or 便名
	VoyageNo       string    // 航海番号 or フライト番号
}

func (p ShippedPayload) milestoneType() MilestoneType {
	return MilestoneShipped
}

// CustomsClearedPayload: 通関書類から確定する事実
type CustomsClearedPayload struct {
	DeclarationNo string
	ClearanceDate time.Time
	Direction     shared.TradeDirection // EXPORT or IMPORT
}

func (p CustomsClearedPayload) milestoneType() MilestoneType {
	if p.Direction == shared.DirectionExport {
		return MilestoneCustomsExportCleared
	}
	return MilestoneCustomsImportCleared
}

// ArrivedPayload: Arrival Notice から確定する事実
type ArrivedPayload struct {
	ArrivalDate   time.Time
	DischargePort string
}

func (p ArrivedPayload) milestoneType() MilestoneType {
	return MilestoneArrived
}

// DeliveredPayload: Delivery Order から確定する事実
type DeliveredPayload struct {
	DeliveryDate     time.Time
	DeliveryLocation string
	ReceiverName     string
}

func (p DeliveredPayload) milestoneType() MilestoneType {
	return MilestoneDelivered
}

// InvoiceReceivedPayload: Invoice から確定する事実
type InvoiceReceivedPayload struct {
	InvoiceNo   string
	InvoiceDate time.Time
	TotalAmount shared.Money
}

func (p InvoiceReceivedPayload) milestoneType() MilestoneType {
	return MilestoneInvoiceReceived
}

// GenericPayload: カスタムマイルストーン用の汎用ペイロード
type GenericPayload struct {
	MType MilestoneType
	Data  map[string]interface{}
}

func (p GenericPayload) milestoneType() MilestoneType {
	return p.MType
}

// ==========================================
// ShipmentExecution: マイルストーンの蓄積管理
// ==========================================

// ShipmentExecution: 出荷実行コンテキスト (Entity)
// 書類を通じて確定した業務的事実をマイルストーンとして蓄積する。
type ShipmentExecution struct {
	milestones []Milestone
}

// newShipmentExecution: ShipmentExecutionのファクトリ
func newShipmentExecution() ShipmentExecution {
	return ShipmentExecution{
		milestones: make([]Milestone, 0),
	}
}

// RecordMilestone: マイルストーンを記録する
// 同一MilestoneTypeでも複数記録可能（履歴保持）。
func (e *ShipmentExecution) RecordMilestone(
	milestoneType MilestoneType,
	occurredAt time.Time,
	sourceDocumentID uuid.UUID,
	sourceDocType shared.DocType,
	payload MilestonePayload,
) (*Milestone, error) {
	if milestoneType == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "milestone type is required")
	}
	if sourceDocumentID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "source document ID is required")
	}
	if payload == nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "payload is required")
	}

	// 同一Type内のSequenceを計算
	seq := 1
	for _, m := range e.milestones {
		if m.Type == milestoneType {
			seq++
		}
	}

	milestone := Milestone{
		ID:               uuid.New(),
		Type:             milestoneType,
		OccurredAt:       occurredAt,
		RecordedAt:       time.Now(),
		SourceDocumentID: sourceDocumentID,
		SourceDocType:    sourceDocType,
		Payload:          payload,
		Sequence:         seq,
	}

	e.milestones = append(e.milestones, milestone)

	return &milestone, nil
}

// Milestones: 全マイルストーンのコピーを返却
func (e *ShipmentExecution) Milestones() []Milestone {
	result := make([]Milestone, len(e.milestones))
	copy(result, e.milestones)
	return result
}

// FindLatestMilestone: 指定MilestoneTypeの最新（最大Sequence）のマイルストーンを取得
func (e *ShipmentExecution) FindLatestMilestone(t MilestoneType) *Milestone {
	var latest *Milestone
	for i := range e.milestones {
		if e.milestones[i].Type == t {
			latest = &e.milestones[i]
		}
	}
	if latest == nil {
		return nil
	}
	// コピーを返す
	copied := *latest
	return &copied
}

// FindMilestonesByType: 指定MilestoneTypeの全マイルストーンを時系列で取得
func (e *ShipmentExecution) FindMilestonesByType(t MilestoneType) []Milestone {
	var result []Milestone
	for _, m := range e.milestones {
		if m.Type == t {
			result = append(result, m)
		}
	}
	return result
}

// HasMilestone: 指定MilestoneTypeが記録済みかどうか
func (e *ShipmentExecution) HasMilestone(t MilestoneType) bool {
	for _, m := range e.milestones {
		if m.Type == t {
			return true
		}
	}
	return false
}

// MilestoneCount: マイルストーンの総数
func (e *ShipmentExecution) MilestoneCount() int {
	return len(e.milestones)
}
