package commercial

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ServiceContractRepository: ServiceContract集約のリポジトリインターフェース
type ServiceContractRepository interface {
	// Save: ServiceContractを保存（新規作成または更新）
	Save(ctx context.Context, contract *ServiceContract) error

	// FindByID: IDでServiceContractを取得
	FindByID(ctx context.Context, id uuid.UUID) (*ServiceContract, error)

	// FindByProviderAndShipper: プロバイダーと荷主の組み合わせで契約を検索
	FindByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID) ([]*ServiceContract, error)

	// FindDraftByProviderAndShipper: DRAFT状態の契約を検索
	// 入札プロセスで既存のDRAFT契約を探す際に使用
	FindDraftByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID) ([]*ServiceContract, error)

	// FindActiveByProviderAndShipper: CONTRACTED状態の有効な契約を検索
	// 料金計算時に使用する契約を探す際に使用
	FindActiveByProviderAndShipper(ctx context.Context, providerID, shipperID uuid.UUID, effectiveDate time.Time) ([]*ServiceContract, error)

	// Delete: ServiceContractを削除（論理削除）
	Delete(ctx context.Context, id uuid.UUID) error
}

// LogisticsProviderRepository: LogisticsProvider集約のリポジトリインターフェース
type LogisticsProviderRepository interface {
	// FindByID: IDでLogisticsProviderを取得
	FindByID(ctx context.Context, id uuid.UUID) (*LogisticsProvider, error)

	// FindByName: 名前でLogisticsProviderを検索（曖昧検索）
	FindByName(ctx context.Context, name string) ([]*LogisticsProvider, error)

	// Save: LogisticsProviderを保存
	Save(ctx context.Context, provider *LogisticsProvider) error
}

// TariffRepository: Tariff集約のリポジトリインターフェース
type TariffRepository interface {
	// Save: Tariffを保存（新規作成または更新）
	Save(ctx context.Context, tariff *Tariff) error

	// FindByID: IDでTariffを取得
	FindByID(ctx context.Context, id uuid.UUID) (*Tariff, error)

	// FindByContractID: 契約IDに紐づくTariffを全件取得
	FindByContractID(ctx context.Context, contractID uuid.UUID) ([]*Tariff, error)

	// FindByContractIDAndName: 契約IDと名前でTariffを検索
	FindByContractIDAndName(ctx context.Context, contractID uuid.UUID, name string) ([]*Tariff, error)

	// FindLatestVersionByContractIDAndName: 契約IDと名前で最新バージョンのTariffを取得
	FindLatestVersionByContractIDAndName(ctx context.Context, contractID uuid.UUID, name string) (*Tariff, error)

	// CountByContractID: 契約IDに紐づくTariffの件数を取得
	CountByContractID(ctx context.Context, contractID uuid.UUID) (int, error)

	// Delete: Tariffを削除
	Delete(ctx context.Context, id uuid.UUID) error
}
