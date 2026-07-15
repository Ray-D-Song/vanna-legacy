package engine

import (
	"testing"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

func TestRecommendChartBar(t *testing.T) {
	data := &domain.QueryResult{
		Columns: []domain.Column{{Name: "name", Type: "string"}, {Name: "sales", Type: "number"}},
		Rows:    [][]any{{"a", 1}, {"b", 2}},
	}
	spec := RecommendChart(data)
	if spec.Type != domain.ChartTypeBar {
		t.Fatalf("expected bar, got %s", spec.Type)
	}
}
