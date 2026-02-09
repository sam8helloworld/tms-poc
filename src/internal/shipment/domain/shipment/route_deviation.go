package shipment

import (
	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/network/domain/route"
)

// ==========================================
// Route Deviation Analysis (ルート逸脱分析)
// ==========================================

// DeviationType: 逸脱タイプ
type DeviationType string

const (
	DeviationMatched         DeviationType = "MATCHED"           // 計画通り
	DeviationLocationChanged DeviationType = "LOCATION_CHANGED"  // 経由地変更
	DeviationSkipped         DeviationType = "SKIPPED"           // スキップ（未実行）
	DeviationAdded           DeviationType = "ADDED"             // 追加区間
)

// RouteDeviationAnalysis: ルート逸脱分析結果
type RouteDeviationAnalysis struct {
	ShipmentID      ShipmentID
	HasDeviation    bool
	DeviationReason string
	SegmentMappings []SegmentMapping
	MissingSegments []route.RouteSegmentID // 計画にあるが実績がない
	ExtraSegments   []uuid.UUID            // 実績にあるが計画にない
}

// SegmentMapping: 計画セグメントと実績セグメントの対応関係
type SegmentMapping struct {
	PlannedSegmentID route.RouteSegmentID
	ActualSegmentID  *uuid.UUID    // nilの場合は未実行
	IsMatched        bool          // 発着地が一致するか
	DeviationType    DeviationType // MATCHED, LOCATION_CHANGED, SKIPPED, etc.
}
