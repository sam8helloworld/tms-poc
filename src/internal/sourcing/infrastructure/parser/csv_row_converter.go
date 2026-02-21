package parser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
	"github.com/shopspring/decimal"
)

// RowConverter: カラムマッピングを使用してCSV行をParsedLineItemに変換
type RowConverter struct {
	analysis *CSVAnalysisResult
	resolver LocationResolver
}

// NewRowConverter: RowConverterを生成
func NewRowConverter(analysis *CSVAnalysisResult, resolver LocationResolver) *RowConverter {
	return &RowConverter{
		analysis: analysis,
		resolver: resolver,
	}
}

// ConvertAll: 全データ行をParsedTariffDataに変換
func (c *RowConverter) ConvertAll(ctx context.Context, allRows [][]string) (*pricing.ParsedTariffData, error) {
	effectiveFrom, err := parseDate(c.analysis.EffectiveFrom)
	if err != nil {
		effectiveFrom = time.Now()
	}
	effectiveTo, err := parseDate(c.analysis.EffectiveTo)
	if err != nil {
		effectiveTo = effectiveFrom.AddDate(1, 0, 0)
	}

	dataRows := allRows
	if c.analysis.DataStartRow < len(allRows) {
		dataRows = allRows[c.analysis.DataStartRow:]
	}

	// 重量帯パターンの検出
	weightMinIdx := c.analysis.RoleIndex(RoleWeightMin)
	weightMaxIdx := c.analysis.RoleIndex(RoleWeightMax)
	isWeightBand := weightMinIdx >= 0 && weightMaxIdx >= 0

	var lineItems []pricing.ParsedLineItem
	if isWeightBand {
		// 重量帯別料金 → ExpressionStrategyにグルーピング
		grouped, err := c.convertWeightBandRows(ctx, dataRows, weightMinIdx, weightMaxIdx)
		if err != nil {
			return nil, fmt.Errorf("convert weight band rows: %w", err)
		}
		lineItems = grouped
	} else {
		// 通常行 → FlatStrategy
		for i, row := range dataRows {
			if isEmptyRow(row) {
				continue
			}
			item, err := c.convertRow(ctx, row)
			if err != nil {
				return nil, fmt.Errorf("row %d: %w", c.analysis.DataStartRow+i+1, err)
			}
			lineItems = append(lineItems, *item)
		}
	}

	return &pricing.ParsedTariffData{
		TariffName:    c.analysis.TariffName,
		EffectiveFrom: effectiveFrom,
		EffectiveTo:   effectiveTo,
		LineItems:     lineItems,
	}, nil
}

// convertRow: 1行をParsedLineItemに変換（FlatStrategy）
func (c *RowConverter) convertRow(ctx context.Context, row []string) (*pricing.ParsedLineItem, error) {
	chargeCode := c.getField(row, RoleChargeCode)
	chargeName := c.getField(row, RoleChargeName)
	if chargeCode == "" && chargeName != "" {
		chargeCode = normalizeChargeCode(chargeName)
	}
	if chargeCode == "" {
		return nil, fmt.Errorf("charge_code is empty")
	}

	category := c.getFieldWithDefault(row, RoleCategory, c.analysis.DefaultCategory)
	if category == "" {
		category = inferCategory(chargeCode)
	}

	amountStr := c.getField(row, RoleAmount)
	amount, err := parseAmount(amountStr)
	if err != nil {
		return nil, fmt.Errorf("parse amount %q: %w", amountStr, err)
	}

	currency := c.getFieldWithDefault(row, RoleCurrency, c.analysis.DefaultCurrency)
	if currency == "" {
		currency = "USD"
	}
	currency = strings.ToUpper(currency)

	// Scope判定
	scopeType, scopeAttrs, err := c.resolveScope(ctx, row)
	if err != nil {
		return nil, fmt.Errorf("resolve scope: %w", err)
	}

	item := &pricing.ParsedLineItem{
		ChargeCode:        strings.ToUpper(chargeCode),
		ChargeName:        chargeName,
		Category:          strings.ToUpper(category),
		ServiceScopeType:  scopeType,
		ServiceScopeAttrs: scopeAttrs,
		PricingType:       pricing.PricingFlat,
		PricingAttrs: map[string]any{
			"Amount":   amount.String(),
			"Currency": currency,
		},
	}

	return item, nil
}

// bandEntry: 重量帯別料金の1エントリ
type bandEntry struct {
	weightMin decimal.Decimal
	weightMax decimal.Decimal
	amount    decimal.Decimal
	currency  string
}

// convertWeightBandRows: 重量帯別行をExpressionStrategyに変換
// 同じchargeCode+route の行をグルーピングし、条件分岐式を生成
func (c *RowConverter) convertWeightBandRows(ctx context.Context, rows [][]string, weightMinIdx, weightMaxIdx int) ([]pricing.ParsedLineItem, error) {
	type groupKey struct {
		chargeCode string
		scopeKey   string
	}

	type group struct {
		key        groupKey
		bands      []bandEntry
		chargeName string
		category   string
		scopeType  pricing.ServiceScopeType
		scopeAttrs map[string]string
	}

	groups := make(map[groupKey]*group)
	var order []groupKey

	for _, row := range rows {
		if isEmptyRow(row) {
			continue
		}

		chargeCode := c.getField(row, RoleChargeCode)
		chargeName := c.getField(row, RoleChargeName)
		if chargeCode == "" && chargeName != "" {
			chargeCode = normalizeChargeCode(chargeName)
		}
		// 重量帯料金は charge_code/name 列がない場合（航空料金等）tariff_name をフォールバックとして使う
		if chargeCode == "" {
			chargeCode = normalizeChargeCode(c.analysis.TariffName)
		}
		if chargeCode == "" {
			chargeCode = "FREIGHT"
		}

		scopeType, scopeAttrs, err := c.resolveScope(ctx, row)
		if err != nil {
			return nil, err
		}

		scopeKey := fmt.Sprintf("%s:%v", scopeType, scopeAttrs)
		gk := groupKey{chargeCode: strings.ToUpper(chargeCode), scopeKey: scopeKey}

		amountStr := c.getField(row, RoleAmount)
		if amountStr == "" {
			// amount列が存在しない行（メタデータ行や幻のカラムマッピングによる空値）はスキップ
			continue
		}
		amount, err := parseAmount(amountStr)
		if err != nil {
			// パース不能な行もスキップ（エラーにしない）
			continue
		}

		wMin := getDecimalField(row, weightMinIdx)
		wMax := getDecimalField(row, weightMaxIdx)

		currency := c.getFieldWithDefault(row, RoleCurrency, c.analysis.DefaultCurrency)
		if currency == "" {
			currency = "USD"
		}

		g, ok := groups[gk]
		if !ok {
			category := c.getFieldWithDefault(row, RoleCategory, c.analysis.DefaultCategory)
			if category == "" {
				category = inferCategory(chargeCode)
			}
			g = &group{
				key:        gk,
				chargeName: chargeName,
				category:   strings.ToUpper(category),
				scopeType:  scopeType,
				scopeAttrs: scopeAttrs,
			}
			groups[gk] = g
			order = append(order, gk)
		}
		g.bands = append(g.bands, bandEntry{
			weightMin: wMin,
			weightMax: wMax,
			amount:    amount,
			currency:  strings.ToUpper(currency),
		})
	}

	var items []pricing.ParsedLineItem
	for _, gk := range order {
		g := groups[gk]
		formula := buildWeightBandFormula(g.bands)

		currency := "USD"
		if len(g.bands) > 0 {
			currency = g.bands[0].currency
		}

		items = append(items, pricing.ParsedLineItem{
			ChargeCode:        gk.chargeCode,
			ChargeName:        g.chargeName,
			Category:          g.category,
			ServiceScopeType:  g.scopeType,
			ServiceScopeAttrs: g.scopeAttrs,
			PricingType:       pricing.PricingExpression,
			PricingAttrs: map[string]any{
				"Formula":  formula,
				"Currency": currency,
			},
		})
	}

	return items, nil
}

// buildWeightBandFormula: 重量帯を条件分岐式に変換
// 例: weight <= 45 ? weight * 8.50 : weight <= 100 ? weight * 7.20 : weight * 6.00
func buildWeightBandFormula(bands []bandEntry) string {
	if len(bands) == 0 {
		return "0"
	}
	if len(bands) == 1 {
		return "weight * " + bands[0].amount.String()
	}

	var parts []string
	for i, b := range bands {
		if i == len(bands)-1 {
			// 最後のバンド（else節）
			parts = append(parts, "weight * "+b.amount.String())
		} else {
			parts = append(parts, fmt.Sprintf("weight <= %s ? weight * %s", b.weightMax.String(), b.amount.String()))
		}
	}
	return strings.Join(parts, " : ")
}

// resolveScope: 行データからServiceScopeを判定
func (c *RowConverter) resolveScope(ctx context.Context, row []string) (pricing.ServiceScopeType, map[string]string, error) {
	originStr := c.getFieldWithDefault(row, RoleOrigin, c.analysis.DefaultOrigin)
	destStr := c.getFieldWithDefault(row, RoleDestination, c.analysis.DefaultDestination)
	locationStr := c.getFieldWithDefault(row, RoleLocation, c.analysis.DefaultLocation)
	modeStr := c.getFieldWithDefault(row, RoleTransportMode, c.analysis.DefaultTransportMode)
	serviceType := c.getFieldWithDefault(row, RoleServiceType, c.analysis.DefaultServiceType)

	attrs := make(map[string]string)

	if originStr != "" && destStr != "" {
		// TRANSPORTATION scope
		originID, err := c.resolver.Resolve(ctx, originStr)
		if err != nil {
			return "", nil, fmt.Errorf("resolve origin %q: %w", originStr, err)
		}
		destID, err := c.resolver.Resolve(ctx, destStr)
		if err != nil {
			return "", nil, fmt.Errorf("resolve destination %q: %w", destStr, err)
		}

		attrs["OriginID"] = uuid.UUID(originID).String()
		attrs["DestinationID"] = uuid.UUID(destID).String()
		if modeStr != "" {
			attrs["Mode"] = strings.ToUpper(modeStr)
		}

		return pricing.ScopeTransportation, attrs, nil
	}

	if locationStr != "" {
		// LOCATION scope
		locID, err := c.resolver.Resolve(ctx, locationStr)
		if err != nil {
			return "", nil, fmt.Errorf("resolve location %q: %w", locationStr, err)
		}

		attrs["LocationID"] = uuid.UUID(locID).String()
		if serviceType != "" {
			attrs["ServiceType"] = strings.ToUpper(serviceType)
		}

		return pricing.ScopeLocation, attrs, nil
	}

	// origin==dest (同一地点) の場合もLOCATION scope
	if originStr != "" && destStr == "" {
		locID, err := c.resolver.Resolve(ctx, originStr)
		if err != nil {
			return "", nil, fmt.Errorf("resolve location from origin %q: %w", originStr, err)
		}
		attrs["LocationID"] = uuid.UUID(locID).String()
		if serviceType != "" {
			attrs["ServiceType"] = strings.ToUpper(serviceType)
		}
		return pricing.ScopeLocation, attrs, nil
	}

	return "", nil, fmt.Errorf("cannot determine scope: no origin/destination or location")
}

// getField: 指定Roleのカラム値を取得
func (c *RowConverter) getField(row []string, role ColumnRole) string {
	idx := c.analysis.RoleIndex(role)
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// getFieldWithDefault: 指定Roleのカラム値を取得（デフォルト値付き）
func (c *RowConverter) getFieldWithDefault(row []string, role ColumnRole, defaultVal string) string {
	v := c.getField(row, role)
	if v == "" {
		return defaultVal
	}
	return v
}

// --- ヘルパー関数 ---

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02-Jan-2006",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}

func parseAmount(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "$")
	s = strings.TrimPrefix(s, "\u00a5") // ¥
	s = strings.TrimPrefix(s, "\u20ac") // €
	return decimal.NewFromString(s)
}

func getDecimalField(row []string, idx int) decimal.Decimal {
	if idx < 0 || idx >= len(row) {
		return decimal.Zero
	}
	v := strings.TrimSpace(row[idx])
	d, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func isEmptyRow(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

// normalizeChargeCode: 料金名称からコードを生成
func normalizeChargeCode(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// 括弧内の略語があればそれを使う
	if idx := strings.Index(name, "("); idx >= 0 {
		if end := strings.Index(name[idx:], ")"); end >= 0 {
			code := strings.TrimSpace(name[idx+1 : idx+end])
			if len(code) <= 10 {
				return strings.ToUpper(code)
			}
		}
	}
	// 単語の頭文字を取る
	words := strings.Fields(name)
	if len(words) == 1 {
		return strings.ToUpper(name)
	}
	var code strings.Builder
	for _, w := range words {
		if len(w) > 0 {
			first := strings.ToUpper(w[:1])
			// 前置詞等は除外
			lower := strings.ToLower(w)
			if lower == "per" || lower == "of" || lower == "the" || lower == "a" || lower == "an" || lower == "and" || lower == "or" || lower == "for" {
				continue
			}
			code.WriteString(first)
		}
	}
	return code.String()
}

// inferCategory: ChargeCodeからカテゴリを推定
func inferCategory(code string) string {
	code = strings.ToUpper(code)
	switch {
	case code == "OFR" || code == "AFR" || code == "OCEAN FREIGHT" || code == "AIR FREIGHT" || code == "DRAYAGE":
		return "FREIGHT"
	case code == "THC" || code == "CFS" || code == "WRS" || code == "STORAGE":
		return "LOCAL"
	case code == "DOC" || code == "BL" || code == "AWB":
		return "DOCUMENTATION"
	case strings.Contains(code, "SURCHARGE") || code == "BAF" || code == "CAF" || code == "PSS" || code == "GRI" || code == "EBS":
		return "SURCHARGE"
	case code == "CUSTOMS" || code == "DUTY" || code == "BROKERAGE":
		return "CUSTOMS"
	default:
		return "OTHER"
	}
}

// parseTransportMode: 文字列からTransportModeを正規化（バリデーションは行わない）
func parseTransportMode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch {
	case s == "OCEAN" || s == "SEA" || s == "FCL" || s == "LCL":
		return "OCEAN"
	case s == "AIR":
		return "AIR"
	case s == "TRUCK" || s == "ROAD" || s == "DRAYAGE" || s == "TRUCKING":
		return "TRUCK"
	case s == "RAIL" || s == "RAILWAY":
		return "Railway"
	default:
		return s
	}
}

var _ = parseTransportMode
