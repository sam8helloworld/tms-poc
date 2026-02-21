package parser

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// CSVTariffParser: LLMを使用したCSV料金表パーサー
//
// パイプライン:
//
//	io.Reader → CSVReader → LLMColumnAnalyzer → RowConverter → LocationResolver → ParsedTariffData
type CSVTariffParser struct {
	ctx      context.Context
	analyzer ColumnAnalyzer
	resolver LocationResolver
}

// NewCSVTariffParser: CSVTariffParserを生成
func NewCSVTariffParser(ctx context.Context, analyzer ColumnAnalyzer, resolver LocationResolver) *CSVTariffParser {
	return &CSVTariffParser{
		ctx:      ctx,
		analyzer: analyzer,
		resolver: resolver,
	}
}

// Parse: CSVファイルを解析してParsedTariffDataを返す
func (p *CSVTariffParser) Parse(reader io.Reader) (*pricing.ParsedTariffData, error) {
	// Stage 1: CSV全行読み込み
	allRows, err := readAllCSV(reader)
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if len(allRows) < 2 {
		return nil, fmt.Errorf("CSV has too few rows (need at least header + 1 data row)")
	}

	// Stage 2: LLMによるカラム構造解析
	// ヘッダー候補（最初の行）+ サンプルデータ（最大10行）を送信
	headers := allRows[0]
	sampleEnd := len(allRows)
	if sampleEnd > 11 { // header + 10 data rows
		sampleEnd = 11
	}
	sampleRows := allRows[1:sampleEnd]

	analysis, err := p.analyzer.Analyze(p.ctx, headers, sampleRows)
	if err != nil {
		return nil, fmt.Errorf("analyze CSV structure: %w", err)
	}

	// Stage 3-4: RowConverter + LocationResolver で全行変換
	converter := NewRowConverter(analysis, p.resolver)
	result, err := converter.ConvertAll(p.ctx, allRows)
	if err != nil {
		return nil, fmt.Errorf("convert rows: %w", err)
	}

	return result, nil
}

// SupportedFormats: サポートするファイル形式
func (p *CSVTariffParser) SupportedFormats() []string {
	return []string{"csv"}
}

// readAllCSV: CSVファイルを全行読み込み
func readAllCSV(reader io.Reader) ([][]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1 // 可変長カラムを許可
	csvReader.TrimLeadingSpace = true
	csvReader.LazyQuotes = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}
	return records, nil
}
