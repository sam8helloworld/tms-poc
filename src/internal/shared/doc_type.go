package shared

// DocType: 国際物流で使用される書類の種別
type DocType string

const (
	DocTypeInvoice              DocType = "INVOICE"
	DocTypePackingList          DocType = "PACKING_LIST"
	DocTypeBillOfLading         DocType = "BILL_OF_LADING"
	DocTypeAirWayBill           DocType = "AIR_WAYBILL"
	DocTypeArrivalNotice        DocType = "ARRIVAL_NOTICE"
	DocTypeCustomsDeclaration   DocType = "CUSTOMS_DECLARATION"
	DocTypeCertificateOfOrigin  DocType = "CERTIFICATE_OF_ORIGIN"
	DocTypeInsuranceCertificate DocType = "INSURANCE_CERTIFICATE"
	DocTypeBookingConfirmation  DocType = "BOOKING_CONFIRMATION"
	DocTypeDeliveryOrder        DocType = "DELIVERY_ORDER"
	DocTypeOther                DocType = "OTHER"
)

// TradeDirection: 貿易方向（輸出・輸入）
type TradeDirection string

const (
	DirectionExport TradeDirection = "EXPORT"
	DirectionImport TradeDirection = "IMPORT"
)
