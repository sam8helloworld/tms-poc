package service

import (
	"sort"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/domain/route"
	"github.com/sam8helloworld/tms-poc/internal/domain/shipment"
	"github.com/sam8helloworld/tms-poc/internal/domain/tracking"
)

// ==========================================
// Domain Service: RouteDeviationService
// ==========================================

// RouteDeviationService: 計画と実績のルート逸脱分析を行うドメインサービス
type RouteDeviationService struct{}

// NewRouteDeviationService: コンストラクタ
func NewRouteDeviationService() *RouteDeviationService {
	return &RouteDeviationService{}
}

// AnalyzeDeviation: 計画と実績のルート分析
// TrackingUnitは計画を知らないため、このサービスが計画と実績の突合を行う
func (s *RouteDeviationService) AnalyzeDeviation(
	ship *shipment.Shipment,
	trackingUnits []*tracking.TrackingUnit,
) shipment.RouteDeviationAnalysis {
	analysis := shipment.RouteDeviationAnalysis{
		ShipmentID:      ship.ID,
		SegmentMappings: []shipment.SegmentMapping{},
		MissingSegments: []route.RouteSegmentID{},
		ExtraSegments:   []uuid.UUID{},
	}

	// 実績の全セグメントを時系列で結合
	actualSegments := s.collectAllActualSegments(trackingUnits)

	// 計画セグメントと実績セグメントをマッチング
	for i, plannedSeg := range ship.Plan.PlannedRoute.Segments {
		mapping := s.matchSegment(plannedSeg, actualSegments, i)
		analysis.SegmentMappings = append(analysis.SegmentMappings, mapping)

		if !mapping.IsMatched {
			analysis.HasDeviation = true
			if mapping.DeviationType == shipment.DeviationSkipped {
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
func (s *RouteDeviationService) collectAllActualSegments(
	trackingUnits []*tracking.TrackingUnit,
) []*tracking.TrackingSegment {
	var allSegments []*tracking.TrackingSegment
	for _, unit := range trackingUnits {
		allSegments = append(allSegments, unit.Segments()...)
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
func (s *RouteDeviationService) matchSegment(
	plannedSeg route.RouteSegment,
	actualSegments []*tracking.TrackingSegment,
	expectedIndex int,
) shipment.SegmentMapping {
	// 期待される順序で実績セグメントを探す
	if expectedIndex < len(actualSegments) {
		actual := actualSegments[expectedIndex]

		// 発着地が一致するかチェック
		if s.isLocationMatched(plannedSeg, actual) {
			return shipment.SegmentMapping{
				PlannedSegmentID: plannedSeg.ID,
				ActualSegmentID:  &actual.ID,
				IsMatched:        true,
				DeviationType:    shipment.DeviationMatched,
			}
		}

		// 発着地が異なる場合は逸脱
		return shipment.SegmentMapping{
			PlannedSegmentID: plannedSeg.ID,
			ActualSegmentID:  &actual.ID,
			IsMatched:        false,
			DeviationType:    shipment.DeviationLocationChanged,
		}
	}

	// 実績が見つからない（未実行）
	return shipment.SegmentMapping{
		PlannedSegmentID: plannedSeg.ID,
		ActualSegmentID:  nil,
		IsMatched:        false,
		DeviationType:    shipment.DeviationSkipped,
	}
}

// isLocationMatched: 発着地が一致するか判定
func (s *RouteDeviationService) isLocationMatched(
	planned route.RouteSegment,
	actual *tracking.TrackingSegment,
) bool {
	return uuid.UUID(planned.OriginLocationID) == actual.ActualOriginLocationID &&
		uuid.UUID(planned.DestLocationID) == actual.ActualDestLocationID
}

// findExtraSegments: 計画にない余分なセグメントを検出
func (s *RouteDeviationService) findExtraSegments(
	actualSegments []*tracking.TrackingSegment,
	mappings []shipment.SegmentMapping,
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
