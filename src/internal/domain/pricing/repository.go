package pricing

import (
	"context"

	"github.com/google/uuid"
)

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
