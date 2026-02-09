package contract

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/shared"
)

// Vendor: 契約コンテキストにおける物流企業（契約の主体）
// 契約相手としての企業情報を管理する
type Vendor struct {
	shared.EventRecorder

	ID   uuid.UUID
	Name string
	Type ProviderType // CARRIER, FORWARDER, WAREHOUSE, etc.

	// 契約コンテキストの関心事
	CreditRating    CreditRating      // 信用格付
	PaymentTerms    PaymentTerms      // 支払条件
	PreferredVendor bool              // 優先ベンダーフラグ
	Capabilities    []VendorCapability // 提供可能サービス
	Contacts        []VendorContact    // 商務担当者情報

	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreditRating: 信用格付
type CreditRating string

const (
	CreditRatingAAA CreditRating = "AAA" // 最上位
	CreditRatingAA  CreditRating = "AA"
	CreditRatingA   CreditRating = "A"
	CreditRatingBBB CreditRating = "BBB"
	CreditRatingBB  CreditRating = "BB"
	CreditRatingB   CreditRating = "B"
	CreditRatingCCC CreditRating = "CCC"
	CreditRatingCC  CreditRating = "CC"
	CreditRatingC   CreditRating = "C"
	CreditRatingD   CreditRating = "D" // デフォルト
)

// PaymentTerms: 支払条件
type PaymentTerms struct {
	DaysFromInvoice int    // 請求書発行からの支払日数
	Currency        string // 決済通貨
}

// VendorCapability: ベンダーが提供可能なサービス種別
type VendorCapability struct {
	ServiceType  string   // "OCEAN_FCL", "AIR_EXPRESS", "CUSTOMS_CLEARANCE"
	CoverageArea []string // カバレッジエリア (UN/LOCODE, 国コード等)
}

// VendorContact: 商務担当者情報
type VendorContact struct {
	Name         string
	Role         string // "Sales Manager", "Contract Manager"
	Email        string
	Phone        string
	IsPrimaryPOC bool // 主要連絡先フラグ
}

// NewVendor: Vendorのファクトリ関数
func NewVendor(name string, providerType ProviderType) (*Vendor, error) {
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "vendor name is required")
	}

	now := time.Now()
	return &Vendor{
		ID:              uuid.New(),
		Name:            name,
		Type:            providerType,
		CreditRating:    CreditRatingBBB, // デフォルト
		PreferredVendor: false,
		Capabilities:    make([]VendorCapability, 0),
		Contacts:        make([]VendorContact, 0),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// SetCreditRating: 信用格付を設定
func (v *Vendor) SetCreditRating(rating CreditRating) {
	v.CreditRating = rating
	v.UpdatedAt = time.Now()
}

// SetPaymentTerms: 支払条件を設定
func (v *Vendor) SetPaymentTerms(terms PaymentTerms) error {
	if terms.DaysFromInvoice < 0 {
		return shared.NewDomainError(shared.ErrInvalidArgument, "payment days must be non-negative")
	}
	v.PaymentTerms = terms
	v.UpdatedAt = time.Now()
	return nil
}

// AddCapability: 提供可能サービスを追加
func (v *Vendor) AddCapability(capability VendorCapability) {
	v.Capabilities = append(v.Capabilities, capability)
	v.UpdatedAt = time.Now()
}

// AddContact: 担当者を追加
func (v *Vendor) AddContact(contact VendorContact) {
	v.Contacts = append(v.Contacts, contact)
	v.UpdatedAt = time.Now()
}

// MarkAsPreferred: 優先ベンダーとして設定
func (v *Vendor) MarkAsPreferred() {
	v.PreferredVendor = true
	v.UpdatedAt = time.Now()
}

// UnmarkAsPreferred: 優先ベンダーから解除
func (v *Vendor) UnmarkAsPreferred() {
	v.PreferredVendor = false
	v.UpdatedAt = time.Now()
}

// ==========================================
// 後方互換性のためのエイリアス
// ==========================================

// LogisticsProvider: Vendorのエイリアス（後方互換性）
type LogisticsProvider = Vendor
