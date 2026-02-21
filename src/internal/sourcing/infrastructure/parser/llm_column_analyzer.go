package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ColumnAnalyzer: CSVカラム構造を解析するインターフェース
type ColumnAnalyzer interface {
	Analyze(ctx context.Context, headers []string, sampleRows [][]string) (*CSVAnalysisResult, error)
}

// LLMColumnAnalyzer: LLMを使用したカラム構造解析
type LLMColumnAnalyzer struct {
	client *LLMClient
}

// NewLLMColumnAnalyzer: LLMカラムアナライザーを生成
func NewLLMColumnAnalyzer(client *LLMClient) *LLMColumnAnalyzer {
	return &LLMColumnAnalyzer{client: client}
}

const systemPrompt = `あなたは国際物流の料金表（Tariff）CSVファイルの構造を解析するエキスパートです。

## タスク
与えられたCSVのヘッダーとサンプルデータから、各カラムの役割（ColumnRole）を特定し、JSONで返してください。

## ColumnRole一覧
- charge_code: 料金コード（例: OFR, THC, DOC, CFS）
- charge_name: 料金名称（例: Ocean Freight, Terminal Handling Charge）
- category: 料金カテゴリ（例: FREIGHT, LOCAL, SURCHARGE, DOCUMENTATION）
- origin: 出発地（港名、都市名、UN/LOCODE）
- destination: 到着地（港名、都市名、UN/LOCODE）
- location: 場所（倉庫、港など単一地点。THCや保管料など場所ベースの料金用）
- transport_mode: 輸送モード（OCEAN, AIR, TRUCK, Railway）
- service_type: サービス種別（HANDLING, STORAGE, CUSTOMS等。location scopeの場合）
- amount: 金額（数値）
- currency: 通貨コード（USD, JPY, EUR等）
- tariff_name: 料金表名（メタデータ行にある場合）
- effective_from: 有効開始日
- effective_to: 有効終了日
- weight_min: 重量帯下限（航空貨物の重量帯別料金用、kg）
- weight_max: 重量帯上限（航空貨物の重量帯別料金用、kg）
- operator_vendor: 実際の作業業者名/ID（下請け業者等）
- ignore: 無関係なカラム

## 注意事項
- CSVの先頭にメタデータ行（料金表名、有効期間等）がある場合がある。その場合header_rowとdata_start_rowを適切に設定
- カラムに含まれない情報でCSVから読み取れるもの（例: 全行同一通貨、メタデータ行の料金表名等）はdefault_*フィールドに設定
- origin+destinationの組がある場合はTRANSPORTATION scope、locationのみの場合はLOCATION scope
- 重量帯別料金（weight_min, weight_max列がある）の場合は航空貨物の段階別料金を示す
- charge_code / charge_name に相当する明示的な列がない場合は、サービス種別・車両タイプ・作業内容など最も「料金名称」に近い列を charge_name としてマップすること`

const fewShotExample = `## 例1: FWDの全区間料金表
入力:
ヘッダー: ["Charge Code","Charge Name","Category","Origin","Destination","Mode","Amount","Currency"]
サンプル:
["THC","Terminal Handling Charge","LOCAL","JPTYO","JPTYO","OCEAN","35000","JPY"]
["OFR","Ocean Freight","FREIGHT","JPTYO","USLAX","OCEAN","1200","USD"]

出力:
{"columns":[{"index":0,"name":"Charge Code","role":"charge_code"},{"index":1,"name":"Charge Name","role":"charge_name"},{"index":2,"name":"Category","role":"category"},{"index":3,"name":"Origin","role":"origin"},{"index":4,"name":"Destination","role":"destination"},{"index":5,"name":"Mode","role":"transport_mode"},{"index":6,"name":"Amount","role":"amount"},{"index":7,"name":"Currency","role":"currency"}],"header_row":0,"data_start_row":1,"tariff_name":"","effective_from":"","effective_to":"","default_transport_mode":"","default_currency":"","default_category":"","default_service_type":"","default_origin":"","default_destination":"","default_location":""}

## 例2: 倉庫業者の料金表
入力:
ヘッダー: ["Service","Rate (JPY)"]
サンプル:
["Storage (per CBM/day)","150"]
["Loading/Unloading","5000"]

出力:
{"columns":[{"index":0,"name":"Service","role":"charge_name"},{"index":1,"name":"Rate (JPY)","role":"amount"}],"header_row":0,"data_start_row":1,"tariff_name":"","effective_from":"","effective_to":"","default_transport_mode":"","default_currency":"JPY","default_category":"LOCAL","default_service_type":"STORAGE","default_origin":"","default_destination":"","default_location":""}

## 例3: ドレージ料金表（明示的なcharge_name列なし）
入力:
ヘッダー: ["Origin","Destination","Vehicle Type","Rate (USD)","Notes"]
サンプル:
["Los Angeles Port","Los Angeles Warehouse","20ft Chassis","350","Standard drayage"]
["Los Angeles Port","Inland Empire DC","40ft Chassis","800","Extended haul"]

出力:
{"columns":[{"index":0,"name":"Origin","role":"origin"},{"index":1,"name":"Destination","role":"destination"},{"index":2,"name":"Vehicle Type","role":"charge_name"},{"index":3,"name":"Rate (USD)","role":"amount"},{"index":4,"name":"Notes","role":"ignore"}],"header_row":0,"data_start_row":1,"tariff_name":"","effective_from":"","effective_to":"","default_transport_mode":"TRUCK","default_currency":"USD","default_category":"","default_service_type":"","default_origin":"","default_destination":"","default_location":""}`

// csvAnalysisSchema: CSVAnalysisResult の JSON Schema（structured outputs用）
var csvAnalysisSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "columns": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "index": { "type": "integer" },
          "name":  { "type": "string" },
          "role": {
            "type": "string",
            "enum": [
              "charge_code", "charge_name", "category",
              "origin", "destination", "location",
              "transport_mode", "service_type",
              "amount", "currency",
              "tariff_name", "effective_from", "effective_to",
              "weight_min", "weight_max",
              "operator_vendor", "ignore"
            ]
          }
        },
        "required": ["index", "name", "role"]
      }
    },
    "header_row":    { "type": "integer" },
    "data_start_row": { "type": "integer" },
    "tariff_name":   { "type": "string" },
    "effective_from": { "type": "string" },
    "effective_to":   { "type": "string" },
    "default_transport_mode": { "type": "string" },
    "default_currency":       { "type": "string" },
    "default_category":       { "type": "string" },
    "default_service_type":   { "type": "string" },
    "default_origin":         { "type": "string" },
    "default_destination":    { "type": "string" },
    "default_location":       { "type": "string" }
  },
  "required": [
    "columns", "header_row", "data_start_row",
    "tariff_name", "effective_from", "effective_to",
    "default_transport_mode", "default_currency", "default_category",
    "default_service_type", "default_origin", "default_destination", "default_location"
  ]
}`)

// csvAnalysisResponseFormat: LLMリクエストに渡すresponse_format
var csvAnalysisResponseFormat = &ResponseFormat{
	Type: "json_schema",
	JSONSchema: JSONSchema{
		Name:   "csv_analysis_result",
		Strict: true,
		Schema: csvAnalysisSchema,
	},
}

// Analyze: CSVのヘッダーとサンプル行からカラムマッピングを解析
func (a *LLMColumnAnalyzer) Analyze(ctx context.Context, headers []string, sampleRows [][]string) (*CSVAnalysisResult, error) {
	userPrompt := buildUserPrompt(headers, sampleRows)

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt + "\n\n" + fewShotExample},
		{Role: "user", Content: userPrompt},
	}

	// 1回目の試行
	result, err := a.tryAnalyze(ctx, messages)
	if err != nil {
		// JSONパース失敗時に1回リトライ
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: "JSONのパースに失敗しました。有効なJSONのみを出力してください。マークダウンのコードブロック(```)は使わないでください。",
		})
		result, err = a.tryAnalyze(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("LLM analysis failed after retry: %w", err)
		}
	}

	return result, nil
}

func (a *LLMColumnAnalyzer) tryAnalyze(ctx context.Context, messages []ChatMessage) (*CSVAnalysisResult, error) {
	content, err := a.client.Complete(ctx, messages, csvAnalysisResponseFormat)
	if err != nil {
		return nil, err
	}

	var result CSVAnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse LLM response as JSON: %w (response: %s)", err, content)
	}

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("invalid analysis result: %w", err)
	}

	return &result, nil
}

// buildUserPrompt: ヘッダーとサンプル行からユーザープロンプトを構築
func buildUserPrompt(headers []string, sampleRows [][]string) string {
	var sb strings.Builder
	sb.WriteString("以下のCSVの構造を解析してください。\n\n")
	sb.WriteString("ヘッダー: ")
	sb.WriteString(formatRow(headers))
	sb.WriteString("\n\nサンプルデータ:\n")

	for i, row := range sampleRows {
		if i >= 10 { // 最大10行
			break
		}
		sb.WriteString(formatRow(row))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatRow: 行データをJSON配列形式の文字列に変換
func formatRow(row []string) string {
	quoted := make([]string, len(row))
	for i, v := range row {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
