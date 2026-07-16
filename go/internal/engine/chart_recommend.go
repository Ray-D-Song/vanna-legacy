package engine

import (
	"fmt"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

func RecommendChart(data *domain.QueryResult) domain.ChartSpec {
	if data == nil || data.IsEmpty() {
		return domain.ChartSpec{Type: domain.ChartTypeTable, Title: "Results"}
	}

	nums := numericColumns(data)
	cats := categoricalColumns(data)

	switch {
	case len(data.Rows) == 1 && len(nums) == 1 && len(cats) == 0:
		return domain.ChartSpec{
			Type:       domain.ChartTypeMetric,
			Title:      nums[0].Name,
			ValueField: nums[0].Name,
			ValueLabel: nums[0].Name,
		}
	case len(nums) >= 2:
		return domain.ChartSpec{
			Type:  domain.ChartTypeScatter,
			Title: "Scatter",
			X:     &domain.FieldRef{Field: nums[0].Name, Label: nums[0].Name},
			Y:     &domain.FieldRef{Field: nums[1].Name, Label: nums[1].Name},
		}
	case len(nums) == 1 && len(cats) >= 1:
		return domain.ChartSpec{
			Type:  domain.ChartTypeBar,
			Title: "Bar Chart",
			X:     &domain.FieldRef{Field: cats[0].Name, Label: cats[0].Name},
			Y:     &domain.FieldRef{Field: nums[0].Name, Label: nums[0].Name},
		}
	case len(cats) >= 1 && cardinality(data, cats[0].Name) <= 8:
		return domain.ChartSpec{
			Type:  domain.ChartTypePie,
			Title: "Distribution",
			X:     &domain.FieldRef{Field: cats[0].Name, Label: cats[0].Name},
		}
	default:
		return domain.ChartSpec{Type: domain.ChartTypeTable, Title: "Results"}
	}
}

func numericColumns(data *domain.QueryResult) []domain.Column {
	var out []domain.Column
	for _, c := range data.Columns {
		if c.Type == "number" {
			out = append(out, c)
		}
	}
	return out
}

func categoricalColumns(data *domain.QueryResult) []domain.Column {
	var out []domain.Column
	for _, c := range data.Columns {
		if c.Type == "string" || c.Type == "datetime" || c.Type == "boolean" || c.Type == "unknown" {
			out = append(out, c)
		}
	}
	return out
}

func cardinality(data *domain.QueryResult, field string) int {
	idx := -1
	for i, c := range data.Columns {
		if c.Name == field {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0
	}
	seen := map[string]struct{}{}
	for _, row := range data.Rows {
		if idx < len(row) {
			seen[fmt.Sprint(row[idx])] = struct{}{}
		}
	}
	return len(seen)
}
