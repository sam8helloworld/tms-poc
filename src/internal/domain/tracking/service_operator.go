package tracking

import (
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/shared"
)

// ServiceOperator: 実行コンテキストにおける物流企業（業務の遂行者）
// 実務担当者、連絡先、実行役割に焦点を当てたモデル
// TrackingUnitから参照される
type ServiceOperator struct {
	ProviderID uuid.UUID // 契約コンテキストのVendor.IDと紐づく

	// 実行コンテキストの関心事
	Name string
	Role OperatorRole // この業者の具体的な役割

	// 実務担当者情報
	OperationalContacts []OperationalContact

	// 実行品質
	PerformanceMetrics PerformanceMetrics

	// システム連携
	IntegrationChannels []IntegrationChannel
}

// OperatorRole: 実行における業者の役割
type OperatorRole string

const (
	RoleTransporter    OperatorRole = "TRANSPORTER"     // 運送業者（海上、航空、陸上）
	RoleWarehouse      OperatorRole = "WAREHOUSE"       // 倉庫業者
	RoleCustomsBroker  OperatorRole = "CUSTOMS_BROKER"  // 通関業者
	RoleDeliveryAgent  OperatorRole = "DELIVERY_AGENT"  // 配送業者
	RolePackingService OperatorRole = "PACKING_SERVICE" // 梱包業者
	RoleInspector      OperatorRole = "INSPECTOR"       // 検査業者
)

// OperationalContact: 実務担当者情報
type OperationalContact struct {
	Name         string
	Role         string // "Operations Manager", "Dispatcher", "Driver"
	Email        string
	Phone        string
	MobilePhone  string
	Available24x7 bool
	Languages    []string
	IsPrimaryPOC bool // 主要連絡先フラグ
}

// PerformanceMetrics: 実行品質メトリクス
type PerformanceMetrics struct {
	OnTimeDeliveryRate float64   // 定時配送率 (%)
	AverageResponseTime int      // 平均応答時間（分）
	ExceptionRate      float64   // 例外発生率 (%)
	LastUpdated        time.Time
}

// IntegrationChannel: システム連携チャネル
type IntegrationChannel struct {
	Type        IntegrationChannelType
	Endpoint    string // API endpoint, EDI接続情報等
	IsActive    bool
	Credentials string // 認証情報（暗号化前提）
}

// IntegrationChannelType: 連携チャネル種別
type IntegrationChannelType string

const (
	ChannelAPI       IntegrationChannelType = "API"        // REST API
	ChannelEDI       IntegrationChannelType = "EDI"        // EDI連携
	ChannelWebhook   IntegrationChannelType = "WEBHOOK"    // Webhook
	ChannelEmail     IntegrationChannelType = "EMAIL"      // メール通知
	ChannelDriverApp IntegrationChannelType = "DRIVER_APP" // ドライバーアプリ
	ChannelManual    IntegrationChannelType = "MANUAL"     // 手動入力
)

// NewServiceOperator: ServiceOperatorのファクトリ関数
func NewServiceOperator(providerID uuid.UUID, name string, role OperatorRole) (*ServiceOperator, error) {
	if providerID == uuid.Nil {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "providerID is required")
	}
	if name == "" {
		return nil, shared.NewDomainError(shared.ErrInvalidArgument, "operator name is required")
	}

	return &ServiceOperator{
		ProviderID:          providerID,
		Name:                name,
		Role:                role,
		OperationalContacts: make([]OperationalContact, 0),
		PerformanceMetrics: PerformanceMetrics{
			OnTimeDeliveryRate: 95.0, // デフォルト
			AverageResponseTime: 30,
			ExceptionRate:      2.0,
			LastUpdated:        time.Now(),
		},
		IntegrationChannels: make([]IntegrationChannel, 0),
	}, nil
}

// AddOperationalContact: 実務担当者を追加
func (so *ServiceOperator) AddOperationalContact(contact OperationalContact) {
	so.OperationalContacts = append(so.OperationalContacts, contact)
}

// GetPrimaryContact: 主要連絡先を取得
func (so *ServiceOperator) GetPrimaryContact() *OperationalContact {
	for i := range so.OperationalContacts {
		if so.OperationalContacts[i].IsPrimaryPOC {
			return &so.OperationalContacts[i]
		}
	}
	if len(so.OperationalContacts) > 0 {
		return &so.OperationalContacts[0]
	}
	return nil
}

// AddIntegrationChannel: システム連携チャネルを追加
func (so *ServiceOperator) AddIntegrationChannel(channel IntegrationChannel) {
	so.IntegrationChannels = append(so.IntegrationChannels, channel)
}

// GetActiveIntegrationChannel: アクティブな連携チャネルを取得
func (so *ServiceOperator) GetActiveIntegrationChannel(channelType IntegrationChannelType) *IntegrationChannel {
	for i := range so.IntegrationChannels {
		if so.IntegrationChannels[i].Type == channelType && so.IntegrationChannels[i].IsActive {
			return &so.IntegrationChannels[i]
		}
	}
	return nil
}

// UpdatePerformanceMetrics: 実行品質メトリクスを更新
func (so *ServiceOperator) UpdatePerformanceMetrics(metrics PerformanceMetrics) {
	so.PerformanceMetrics = metrics
	so.PerformanceMetrics.LastUpdated = time.Now()
}

// IsReliable: 信頼性の高い業者か判定（定時配送率90%以上、例外率5%以下）
func (so *ServiceOperator) IsReliable() bool {
	return so.PerformanceMetrics.OnTimeDeliveryRate >= 90.0 &&
		so.PerformanceMetrics.ExceptionRate <= 5.0
}

// SupportsRealTimeTracking: リアルタイム追跡をサポートしているか
func (so *ServiceOperator) SupportsRealTimeTracking() bool {
	for _, channel := range so.IntegrationChannels {
		if !channel.IsActive {
			continue
		}
		if channel.Type == ChannelAPI || channel.Type == ChannelWebhook || channel.Type == ChannelDriverApp {
			return true
		}
	}
	return false
}
