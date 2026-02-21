package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
	networkpersistence "github.com/sam8helloworld/tms-poc/internal/network/infrastructure/persistence"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/infrastructure/parser"
)

// CsvTariffParseScenario: CSVTariffParserの動作確認シナリオ
// testdataのCSVファイルをパースし、ParsedTariffData の内容を表形式で出力する
type CsvTariffParseScenario struct{}

func (s *CsvTariffParseScenario) Name() string { return "csv-tariff-parse" }
func (s *CsvTariffParseScenario) Description() string {
	return "CSV tariff parser demo: parse 4 CSV files → display ParsedTariffData"
}

func (s *CsvTariffParseScenario) Run(ctx context.Context, deps *ScenarioDeps, pool *pgxpool.Pool) error {
	fmt.Println("=== CSV Tariff Parse Scenario ===")
	fmt.Println()

	// === Setup: Location master ===
	if err := s.setupLocations(ctx, pool); err != nil {
		return fmt.Errorf("setup locations: %w", err)
	}

	// testdataディレクトリのパスを解決
	testdataDir, err := s.resolveTestdataDir()
	if err != nil {
		return fmt.Errorf("resolve testdata dir: %w", err)
	}

	// LLM設定（LM Studio デフォルト）
	llmEndpoint := envOrDefault("LLM_ENDPOINT", "http://localhost:1234")
	llmModel := envOrDefault("LLM_MODEL", "qwen3-14b")

	fmt.Printf("  LLM: %s (model: %s)\n", llmEndpoint, llmModel)
	fmt.Println()

	// パーサー構築
	llmClient := parser.NewLLMClient(llmEndpoint, llmModel)
	analyzer := parser.NewLLMColumnAnalyzer(llmClient)
	resolver := parser.NewPostgresLocationResolver(pool)
	csvParser := parser.NewCSVTariffParser(ctx, analyzer, resolver)

	// === Step 1: FWD全区間 ===
	if err := s.parseAndDisplay(csvParser, testdataDir, "fwd_full_route.csv", "Step 1", "FWD全区間料金表（OCEAN+TRUCK, JPY/USD）"); err != nil {
		return fmt.Errorf("step 1: %w", err)
	}

	// === Step 2: 乙仲 ===
	if err := s.parseAndDisplay(csvParser, testdataDir, "customs_broker.csv", "Step 2", "乙仲料金表（LOCATIONスコープ）"); err != nil {
		return fmt.Errorf("step 2: %w", err)
	}

	// === Step 3: ドレージ ===
	if err := s.parseAndDisplay(csvParser, testdataDir, "trucking_drayage.csv", "Step 3", "ドレージ料金表（TRUCKモード）"); err != nil {
		return fmt.Errorf("step 3: %w", err)
	}

	// === Step 4: 航空 ===
	if err := s.parseAndDisplay(csvParser, testdataDir, "airline_air_freight.csv", "Step 4", "航空料金表（ExpressionStrategy）"); err != nil {
		return fmt.Errorf("step 4: %w", err)
	}

	fmt.Println()
	fmt.Println("=== Scenario Complete ===")
	return nil
}

// setupLocations: CSVに登場する拠点を登録
func (s *CsvTariffParseScenario) setupLocations(ctx context.Context, pool *pgxpool.Pool) error {
	fmt.Print("[Setup] Creating 11 locations... ")

	repo := networkpersistence.NewPostgresLocationRepo(pool)

	defs := []struct {
		name    string
		code    string
		country string
		locType string
	}{
		{"Tokyo", "JPTYO", "JP", "PORT"},
		{"Tokyo Narita", "JPNRT", "JP", "PORT"},
		{"Yokohama", "JPYOK", "JP", "PORT"},
		{"Los Angeles", "USLAX", "US", "PORT"},
		{"Los Angeles Port", "", "US", "PORT"},
		{"Los Angeles Warehouse", "", "US", "WAREHOUSE"},
		{"Long Beach Port", "", "US", "PORT"},
		{"Inland Empire DC", "", "US", "WAREHOUSE"},
		{"New York", "USNYC", "US", "PORT"},
		{"Felixstowe", "GBFXT", "GB", "PORT"},
		{"Hamburg", "DEHAM", "DE", "PORT"},
	}

	for _, d := range defs {
		var code *string
		if d.code != "" {
			c := d.code
			code = &c
		}
		loc := &route.Location{
			ID:          route.LocationID(uuid.New()),
			Name:        d.name,
			UnLocode:    code,
			CountryCode: d.country,
			Type:        d.locType,
		}
		if err := repo.Save(ctx, loc); err != nil {
			return fmt.Errorf("save location %s: %w", d.name, err)
		}
	}

	fmt.Println("done")

	fmt.Println()
	fmt.Println("  ┌─ [拠点マスタ] 登録済み拠点 ─────────────────────────────")
	fmt.Printf("  │ %-5s  %-25s  %-8s  %s\n", "Code", "Name", "Country", "Type")
	fmt.Println("  │ " + repeatChar('-', 55))
	for _, d := range defs {
		code := "-"
		if d.code != "" {
			code = d.code
		}
		fmt.Printf("  │ %-5s  %-25s  %-8s  %s\n", code, d.name, d.country, d.locType)
	}
	fmt.Printf("  └─ 計 %d 件\n", len(defs))
	fmt.Println()

	return nil
}

// resolveTestdataDir: parser/testdata ディレクトリの絶対パスを解決
func (s *CsvTariffParseScenario) resolveTestdataDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get caller info")
	}
	// scenarios/ → ../../internal/sourcing/infrastructure/parser/testdata
	scenarioDir := filepath.Dir(currentFile)
	testdataDir := filepath.Join(scenarioDir, "..", "..", "..", "..", "internal", "sourcing", "infrastructure", "parser", "testdata")
	absPath, err := filepath.Abs(testdataDir)
	if err != nil {
		return "", fmt.Errorf("resolve abs path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("testdata dir not found: %s", absPath)
	}
	return absPath, nil
}

// parseAndDisplay: CSVをパースして結果を表形式で出力
func (s *CsvTariffParseScenario) parseAndDisplay(p pricing.TariffParser, testdataDir, filename, step, description string) error {
	fmt.Printf("[%s] %s\n", step, description)
	fmt.Printf("  → ファイル: %s\n", filename)

	filePath := filepath.Join(testdataDir, filename)
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", filename, err)
	}
	defer f.Close()

	result, err := p.Parse(f)
	if err != nil {
		return fmt.Errorf("parse %s: %w", filename, err)
	}

	// ヘッダ情報
	fmt.Println()
	fmt.Println("  ┌─ [パース結果] " + strings.Repeat("─", 60))
	fmt.Printf("  │ TariffName:    %s\n", displayOrNA(result.TariffName))
	fmt.Printf("  │ EffectiveFrom: %s\n", displayOrNA(result.EffectiveFrom.Format("2006-01-02")))
	fmt.Printf("  │ EffectiveTo:   %s\n", displayOrNA(result.EffectiveTo.Format("2006-01-02")))
	fmt.Printf("  │ LineItems:     %d 件\n", len(result.LineItems))
	fmt.Println("  │")

	// LineItem テーブル
	fmt.Printf("  │ %-4s %-12s %-14s %-16s %-12s %s\n",
		"#", "ChargeCode", "Category", "ScopeType", "PricingType", "PricingAttrs")
	fmt.Println("  │ " + repeatChar('-', 80))

	for i, item := range result.LineItems {
		chargeCode := item.ChargeCode
		if chargeCode == "" {
			chargeCode = item.ChargeName
		}

		pricingAttrs := formatPricingAttrs(item.PricingType, item.PricingAttrs)
		scopeInfo := formatScopeInfo(item.ServiceScopeType, item.ServiceScopeAttrs)

		fmt.Printf("  │ %-4d %-12s %-14s %-16s %-12s %s\n",
			i+1,
			truncate(chargeCode, 12),
			truncate(item.Category, 14),
			string(item.ServiceScopeType),
			string(item.PricingType),
			truncate(pricingAttrs, 30),
		)
		if scopeInfo != "" {
			fmt.Printf("  │      %s\n", scopeInfo)
		}
	}

	fmt.Printf("  └─ 計 %d 件\n", len(result.LineItems))
	fmt.Println()

	return nil
}

// formatPricingAttrs: PricingAttrsを人間向け文字列に変換
func formatPricingAttrs(pricingType pricing.PricingStrategyType, attrs map[string]any) string {
	switch pricingType {
	case pricing.PricingFlat:
		amount := fmt.Sprintf("%v", attrs["Amount"])
		currency := fmt.Sprintf("%v", attrs["Currency"])
		return amount + " " + currency
	case pricing.PricingExpression:
		formula := fmt.Sprintf("%v", attrs["Formula"])
		currency := fmt.Sprintf("%v", attrs["Currency"])
		return formula + " (" + currency + ")"
	default:
		parts := make([]string, 0, len(attrs))
		for k, v := range attrs {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(parts, ", ")
	}
}

// formatScopeInfo: ServiceScopeAttrsを1行のスコープ情報文字列に変換
func formatScopeInfo(scopeType pricing.ServiceScopeType, attrs map[string]string) string {
	switch scopeType {
	case pricing.ScopeTransportation:
		origin := attrs["OriginID"]
		dest := attrs["DestinationID"]
		mode := attrs["Mode"]
		if origin == "" && dest == "" {
			return ""
		}
		return fmt.Sprintf("→ %s → %s (%s)", shortUUID(origin), shortUUID(dest), mode)
	case pricing.ScopeLocation:
		locID := attrs["LocationID"]
		svcType := attrs["ServiceType"]
		if locID == "" {
			return ""
		}
		return fmt.Sprintf("@ %s [%s]", shortUUID(locID), svcType)
	default:
		return ""
	}
}

// shortUUID: UUIDを先頭8文字に短縮
func shortUUID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// displayOrNA: 空文字列の場合 "N/A" を返す
func displayOrNA(s string) string {
	if s == "" || s == "0001-01-01" {
		return "N/A"
	}
	return s
}

// envOrDefault: 環境変数が未設定の場合デフォルト値を返す
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
