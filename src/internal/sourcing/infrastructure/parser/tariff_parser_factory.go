package parser

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// DefaultTariffParserFactory: TariffParserFactoryのデフォルト実装
type DefaultTariffParserFactory struct {
	ctx          context.Context
	llmEndpoint  string
	llmModel     string
	pool         *pgxpool.Pool
}

// NewDefaultTariffParserFactory: ファクトリを生成
func NewDefaultTariffParserFactory(ctx context.Context, llmEndpoint, llmModel string, pool *pgxpool.Pool) *DefaultTariffParserFactory {
	return &DefaultTariffParserFactory{
		ctx:         ctx,
		llmEndpoint: llmEndpoint,
		llmModel:    llmModel,
		pool:        pool,
	}
}

// GetParser: ファイル形式に応じたパーサーを返す
func (f *DefaultTariffParserFactory) GetParser(format string) (pricing.TariffParser, error) {
	switch format {
	case "csv":
		llmClient := NewLLMClient(f.llmEndpoint, f.llmModel)
		analyzer := NewLLMColumnAnalyzer(llmClient)
		resolver := NewPostgresLocationResolver(f.pool)
		return NewCSVTariffParser(f.ctx, analyzer, resolver), nil
	default:
		return nil, fmt.Errorf("unsupported format: %q (supported: csv)", format)
	}
}
