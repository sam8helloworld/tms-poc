package parser

import "fmt"

// ColumnRole: LLMが各CSVカラムに割り当てる役割
type ColumnRole string

const (
	RoleChargeCode     ColumnRole = "charge_code"
	RoleChargeName     ColumnRole = "charge_name"
	RoleCategory       ColumnRole = "category"
	RoleOrigin         ColumnRole = "origin"
	RoleDestination    ColumnRole = "destination"
	RoleLocation       ColumnRole = "location"
	RoleTransportMode  ColumnRole = "transport_mode"
	RoleServiceType    ColumnRole = "service_type"
	RoleAmount         ColumnRole = "amount"
	RoleCurrency       ColumnRole = "currency"
	RoleTariffName     ColumnRole = "tariff_name"
	RoleEffectiveFrom  ColumnRole = "effective_from"
	RoleEffectiveTo    ColumnRole = "effective_to"
	RoleWeightMin      ColumnRole = "weight_min"
	RoleWeightMax      ColumnRole = "weight_max"
	RoleOperatorVendor ColumnRole = "operator_vendor"
	RoleIgnore         ColumnRole = "ignore"
)

// ColumnMapping: 1カラムのマッピング情報
type ColumnMapping struct {
	Index int        `json:"index"`
	Name  string     `json:"name"`
	Role  ColumnRole `json:"role"`
}

// CSVAnalysisResult: LLMによるCSV構造解析結果
type CSVAnalysisResult struct {
	Columns      []ColumnMapping `json:"columns"`
	HeaderRow    int             `json:"header_row"`
	DataStartRow int             `json:"data_start_row"`

	// CSVメタデータから抽出（カラムに無い場合）
	TariffName    string `json:"tariff_name"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`

	// カラムに無い場合のデフォルト値
	DefaultTransportMode string `json:"default_transport_mode,omitempty"`
	DefaultCurrency      string `json:"default_currency,omitempty"`
	DefaultCategory      string `json:"default_category,omitempty"`
	DefaultServiceType   string `json:"default_service_type,omitempty"`
	DefaultOrigin        string `json:"default_origin,omitempty"`
	DefaultDestination   string `json:"default_destination,omitempty"`
	DefaultLocation      string `json:"default_location,omitempty"`
}

// Validate: 解析結果の妥当性チェック
func (r *CSVAnalysisResult) Validate() error {
	if len(r.Columns) == 0 {
		return fmt.Errorf("no columns mapped")
	}

	hasAmount := false
	hasChargeCode := false
	hasWeightBand := false
	for _, col := range r.Columns {
		if col.Role == RoleAmount {
			hasAmount = true
		}
		if col.Role == RoleChargeCode || col.Role == RoleChargeName {
			hasChargeCode = true
		}
		if col.Role == RoleWeightMin || col.Role == RoleWeightMax {
			hasWeightBand = true
		}
	}

	if !hasAmount {
		return fmt.Errorf("amount column is required")
	}
	// 重量帯別料金表（航空貨物等）は charge_name/charge_code 列が存在しない場合がある
	if !hasChargeCode && !hasWeightBand {
		return fmt.Errorf("charge_code or charge_name column is required")
	}

	// ルート情報: origin+destination または location のいずれかが必要
	hasOrigin := false
	hasDestination := false
	hasLocation := false
	for _, col := range r.Columns {
		switch col.Role {
		case RoleOrigin:
			hasOrigin = true
		case RoleDestination:
			hasDestination = true
		case RoleLocation:
			hasLocation = true
		}
	}
	hasDefaultOrigin := r.DefaultOrigin != ""
	hasDefaultDestination := r.DefaultDestination != ""
	hasDefaultLocation := r.DefaultLocation != ""

	hasRoute := (hasOrigin || hasDefaultOrigin) && (hasDestination || hasDefaultDestination)
	hasLoc := hasLocation || hasDefaultLocation

	if !hasRoute && !hasLoc {
		return fmt.Errorf("route (origin+destination) or location information is required")
	}

	return nil
}

// RoleIndex: 指定Roleの最初のカラムインデックスを返す。見つからない場合は-1
func (r *CSVAnalysisResult) RoleIndex(role ColumnRole) int {
	for _, col := range r.Columns {
		if col.Role == role {
			return col.Index
		}
	}
	return -1
}

// RoleIndices: 指定Roleの全カラムインデックスを返す（複数amount等）
func (r *CSVAnalysisResult) RoleIndices(role ColumnRole) []int {
	var indices []int
	for _, col := range r.Columns {
		if col.Role == role {
			indices = append(indices, col.Index)
		}
	}
	return indices
}
