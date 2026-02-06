package shipment

import (
	"sort"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/tracking"
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

// ==========================================
// Shipmentのドメインメソッド（計画と実績の分析）
// ==========================================

// AnalyzeRouteDeviation: 計画と実績のルート分析
// TrackingUnitは計画を知らないため、Shipmentが計画と実績の突合を行う
func (s *Shipment) AnalyzeRouteDeviation(
	trackingUnits []*tracking.TrackingUnit,
) RouteDeviationAnalysis {
	analysis := RouteDeviationAnalysis{
		ShipmentID:      s.ID,
		SegmentMappings: []SegmentMapping{},
		MissingSegments: []route.RouteSegmentID{},
		ExtraSegments:   []uuid.UUID{},
	}

	// 実績の全セグメントを時系列で結合
	actualSegments := s.collectAllActualSegments(trackingUnits)

	// 計画セグメントと実績セグメントをマッチング
	for i, plannedSeg := range s.Plan.PlannedRoute.Segments {
		mapping := s.matchSegment(plannedSeg, actualSegments, i)
		analysis.SegmentMappings = append(analysis.SegmentMappings, mapping)

		if !mapping.IsMatched {
			analysis.HasDeviation = true
			if mapping.DeviationType == DeviationSkipped {
				analysis.MissingSegments = append(analysis.MissingSegments, plannedSeg.ID)
			}
		}
	}

	// 計画にない余分なセグメントを検出
	analysis.ExtraSegments = s.findExtraSegments(actualSegments, analysis.SegmentMappings)
	if len(analysis.ExtraSegments) > 0 {
		analysis.HasDeviation = true
	}

	return analysis
}

// collectAllActualSegments: 全TrackingUnitの実績セグメントを時系列で結合
func (s *Shipment) collectAllActualSegments(
	trackingUnits []*tracking.TrackingUnit,
) []*tracking.TrackingSegment {
	var allSegments []*tracking.TrackingSegment
	for _, unit := range trackingUnits {
		allSegments = append(allSegments, unit.Segments...)
	}

	// 実際の出発時刻でソート
	sort.Slice(allSegments, func(i, j int) bool {
		if allSegments[i].ActualDeparture == nil {
			return false
		}
		if allSegments[j].ActualDeparture == nil {
			return true
		}
		return allSegments[i].ActualDeparture.Before(*allSegments[j].ActualDeparture)
	})

	return allSegments
}

// matchSegment: 計画セグメントに対応する実績セグメントを探す
func (s *Shipment) matchSegment(
	plannedSeg route.RouteSegment,
	actualSegments []*tracking.TrackingSegment,
	expectedIndex int,
) SegmentMapping {
	// 期待される順序で実績セグメントを探す
	if expectedIndex < len(actualSegments) {
		actual := actualSegments[expectedIndex]

		// 発着地が一致するかチェック
		if s.isLocationMatched(plannedSeg, actual) {
			return SegmentMapping{
				PlannedSegmentID: plannedSeg.ID,
				ActualSegmentID:  &actual.ID,
				IsMatched:        true,
				DeviationType:    DeviationMatched,
			}
		}

		// 発着地が異なる場合は逸脱
		return SegmentMapping{
			PlannedSegmentID: plannedSeg.ID,
			ActualSegmentID:  &actual.ID,
			IsMatched:        false,
			DeviationType:    DeviationLocationChanged,
		}
	}

	// 実績が見つからない（未実行）
	return SegmentMapping{
		PlannedSegmentID: plannedSeg.ID,
		ActualSegmentID:  nil,
		IsMatched:        false,
		DeviationType:    DeviationSkipped,
	}
}

// isLocationMatched: 発着地が一致するか判定
func (s *Shipment) isLocationMatched(
	planned route.RouteSegment,
	actual *tracking.TrackingSegment,
) bool {
	return uuid.UUID(planned.OriginLocationID) == actual.ActualOriginLocationID &&
		uuid.UUID(planned.DestLocationID) == actual.ActualDestLocationID
}

// findExtraSegments: 計画にない余分なセグメントを検出
func (s *Shipment) findExtraSegments(
	actualSegments []*tracking.TrackingSegment,
	mappings []SegmentMapping,
) []uuid.UUID {
	// マッピングされた実績セグメントIDのセット
	mapped := make(map[uuid.UUID]bool)
	for _, m := range mappings {
		if m.ActualSegmentID != nil {
			mapped[*m.ActualSegmentID] = true
		}
	}

	// マッピングされていない実績セグメント = 余分なセグメント
	var extra []uuid.UUID
	for _, seg := range actualSegments {
		if !mapped[seg.ID] {
			extra = append(extra, seg.ID)
		}
	}
	return extra
}
