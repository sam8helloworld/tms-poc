package document

import (
	"time"

	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// ==========================================
// DocumentContent: 書類タイプ別の構造化データ
// ==========================================

// DocumentContent: 書類の構造化コンテンツを表すinterface
// 書類タイプごとに実装を持ち、単体での整合性検証を担う。
type DocumentContent interface {
	// ContentDocType: このコンテンツが対応する書類タイプ
	ContentDocType() shared.DocType
	// Validate: この書類単体としての整合性検証
	Validate() *shared.DomainError
}

// ==========================================
// CommercialInvoiceContent: Commercial Invoiceのコンテンツ
// ==========================================

// CommercialInvoiceContent: Commercial Invoiceの構造化データ
type CommercialInvoiceContent struct {
	InvoiceNo    string
	InvoiceDate  time.Time
	PaymentTerms string
	TotalAmount  shared.Money
	LineItems    []CommercialInvoiceLineItem
	Extension    map[string]interface{} // 各社固有フィールド
}

// CommercialInvoiceLineItem: Commercial Invoiceの明細行
type CommercialInvoiceLineItem struct {
	LineNo      int
	Description string
	Quantity    decimal.Decimal
	UnitPrice   shared.Money
	Amount      shared.Money
	HSCode      string
}

func (c *CommercialInvoiceContent) ContentDocType() shared.DocType {
	return shared.DocTypeCommercialInvoice
}

func (c *CommercialInvoiceContent) Validate() *shared.DomainError {
	if c.InvoiceNo == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "invoice number is required")
	}
	if c.InvoiceDate.IsZero() {
		return shared.NewDomainError(shared.ErrInvalidArgument, "invoice date is required")
	}
	if len(c.LineItems) == 0 {
		return shared.NewDomainError(shared.ErrInvalidArgument, "invoice must have at least one line item")
	}
	return nil
}

// ==========================================
// FreightInvoiceContent: Freight Invoiceのコンテンツ
// ==========================================

// FreightInvoiceContent: Freight Invoiceの構造化データ
type FreightInvoiceContent struct {
	InvoiceNo    string
	InvoiceDate  time.Time
	PaymentTerms string
	TotalAmount  shared.Money
	LineItems    []FreightInvoiceLineItem
	Extension    map[string]interface{}
}

// FreightInvoiceLineItem: Freight Invoiceの明細行
type FreightInvoiceLineItem struct {
	ChargeCode  string
	Description string
	Quantity    decimal.Decimal
	UnitPrice   shared.Money
	Amount      shared.Money
}

func (c *FreightInvoiceContent) ContentDocType() shared.DocType {
	return shared.DocTypeFreightInvoice
}

func (c *FreightInvoiceContent) Validate() *shared.DomainError {
	if c.InvoiceNo == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "invoice number is required")
	}
	if c.InvoiceDate.IsZero() {
		return shared.NewDomainError(shared.ErrInvalidArgument, "invoice date is required")
	}
	if len(c.LineItems) == 0 {
		return shared.NewDomainError(shared.ErrInvalidArgument, "freight invoice must have at least one line item")
	}
	return nil
}

// ==========================================
// PackingListContent: パッキングリストのコンテンツ
// ==========================================

// PackingListContent: パッキングリストの構造化データ
type PackingListContent struct {
	InvoiceNo   string // 対応するInvoice番号
	TotalWeight decimal.Decimal
	TotalVolume decimal.Decimal
	LineItems   []PackingLineItem
	Extension   map[string]interface{}
}

// PackingLineItem: パッキングリストの明細行
type PackingLineItem struct {
	LineNo      int
	Description string
	Quantity    decimal.Decimal
	NetWeight   decimal.Decimal
	GrossWeight decimal.Decimal
	Volume      decimal.Decimal
	PackageType string // Carton, Pallet, etc.
	Marks       string // 荷印
}

func (c *PackingListContent) ContentDocType() shared.DocType {
	return shared.DocTypePackingList
}

func (c *PackingListContent) Validate() *shared.DomainError {
	if len(c.LineItems) == 0 {
		return shared.NewDomainError(shared.ErrInvalidArgument, "packing list must have at least one line item")
	}
	return nil
}

// ==========================================
// BillOfLadingContent: B/Lのコンテンツ
// ==========================================

// BLType: B/Lの種別
type BLType string

const (
	BLTypeOriginal    BLType = "ORIGINAL"
	BLTypeSurrendered BLType = "SURRENDERED"
	BLTypeWaybill     BLType = "WAYBILL"
)

// BillOfLadingContent: B/Lの構造化データ
type BillOfLadingContent struct {
	BLNumber        string
	BLType          BLType
	ShipperName     string
	ConsigneeName   string
	NotifyParty     string
	VesselName      string
	VoyageNo        string
	PortOfLoading   string
	PortOfDischarge string
	OnBoardDate     *time.Time
	Containers      []BLContainerDetail
	Extension       map[string]interface{}
}

// BLContainerDetail: B/Lのコンテナ明細
type BLContainerDetail struct {
	ContainerNo string
	SealNo      string
	PackageCount int
	GrossWeight  decimal.Decimal
	Volume       decimal.Decimal
}

func (c *BillOfLadingContent) ContentDocType() shared.DocType {
	return shared.DocTypeBillOfLading
}

func (c *BillOfLadingContent) Validate() *shared.DomainError {
	if c.BLNumber == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "B/L number is required")
	}
	if c.BLType == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "B/L type is required")
	}
	return nil
}

// ==========================================
// AirWayBillContent: AWBのコンテンツ
// ==========================================

// AirWayBillContent: AWBの構造化データ
type AirWayBillContent struct {
	AWBNumber       string
	ShipperName     string
	ConsigneeName   string
	FlightNo        string
	AirportOfDeparture string
	AirportOfDestination string
	Pieces          int
	GrossWeight     decimal.Decimal
	ChargeableWeight decimal.Decimal
	Extension       map[string]interface{}
}

func (c *AirWayBillContent) ContentDocType() shared.DocType {
	return shared.DocTypeAirWayBill
}

func (c *AirWayBillContent) Validate() *shared.DomainError {
	if c.AWBNumber == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "AWB number is required")
	}
	return nil
}

// ==========================================
// ShippingInstructionContent: S/Iのコンテンツ
// ==========================================

// ShippingInstructionContent: Shipping Instructionの構造化データ
type ShippingInstructionContent struct {
	ShipperName       string
	ConsigneeName     string
	NotifyParty       string
	PortOfLoading     string
	PortOfDischarge   string
	PlaceOfDelivery   string
	RequestedETD      *time.Time
	TransportMode     string
	ContainerType     string
	Commodity         string
	CargoDescription  string
	SpecialInstructions string
	LineItems         []SILineItem
	Extension         map[string]interface{}
}

// SILineItem: S/Iの貨物明細
type SILineItem struct {
	LineNo      int
	Description string
	Quantity    decimal.Decimal
	GrossWeight decimal.Decimal
	Volume      decimal.Decimal
	HSCode      string
}

func (c *ShippingInstructionContent) ContentDocType() shared.DocType {
	return shared.DocTypeShippingInstruction
}

func (c *ShippingInstructionContent) Validate() *shared.DomainError {
	if c.PortOfLoading == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "port of loading is required")
	}
	if c.PortOfDischarge == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "port of discharge is required")
	}
	return nil
}

// ==========================================
// CustomsDeclarationContent: 通関書類のコンテンツ
// ==========================================

// CustomsDeclarationContent: 通関申告書の構造化データ
type CustomsDeclarationContent struct {
	DeclarationNo string
	Direction     shared.TradeDirection // EXPORT or IMPORT
	DeclarantName string
	FilingDate    *time.Time
	ClearanceDate *time.Time
	TotalAmount   *shared.Money // 申告金額
	LineItems     []CustomsLineItem
	Extension     map[string]interface{}
}

// CustomsLineItem: 通関申告の明細行
type CustomsLineItem struct {
	LineNo      int
	HSCode      string
	Description string
	Quantity    decimal.Decimal
	UnitPrice   shared.Money
	Amount      shared.Money
	DutyRate    *decimal.Decimal
	DutyAmount  *shared.Money
}

func (c *CustomsDeclarationContent) ContentDocType() shared.DocType {
	return shared.DocTypeCustomsDeclaration
}

func (c *CustomsDeclarationContent) Validate() *shared.DomainError {
	if c.DeclarationNo == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "declaration number is required")
	}
	if c.Direction == "" {
		return shared.NewDomainError(shared.ErrInvalidArgument, "trade direction is required")
	}
	return nil
}

// ==========================================
// GenericContent: その他書類の汎用コンテンツ
// ==========================================

// GenericContent: 専用構造を持たない書類の汎用コンテンツ
type GenericContent struct {
	DocTypeValue shared.DocType
	Fields       map[string]interface{}
}

func (c *GenericContent) ContentDocType() shared.DocType {
	return c.DocTypeValue
}

func (c *GenericContent) Validate() *shared.DomainError {
	return nil
}
