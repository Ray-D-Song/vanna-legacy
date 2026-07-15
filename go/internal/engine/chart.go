package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

func (s *Service) GenerateChartSpec(ctx context.Context, question, sql string, data *domain.QueryResult) (domain.ChartSpec, error) {
	base := RecommendChart(data)
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: fmt.Sprintf("The user asked: '%s'\nSQL: %s\nData schema:\n%s\n\nReturn ONLY valid JSON for a chart spec.", question, sql, schemaDescription(data))},
		{Role: domain.RoleUser, Content: "Generate a ChartSpec JSON object with fields: type (bar|line|scatter|pie|metric|table), title, x, y, sort, limit. Use column names from the data. No markdown, no explanation."},
	}
	text, err := s.llm.Chat(ctx, messages, s.chatOpts)
	if err != nil {
		return base, err
	}
	spec, err := parseChartSpec(text)
	if err != nil {
		return base, err
	}
	return spec, nil
}

func (s *Service) RefineChartSpec(ctx context.Context, question, sql string, data *domain.QueryResult, base domain.ChartSpec, instructions string) (domain.ChartSpec, error) {
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: fmt.Sprintf("Question: %s\nSQL: %s\nData schema:\n%s\nCurrent chart spec JSON:\n%s", question, sql, schemaDescription(data), mustJSON(base))},
		{Role: domain.RoleUser, Content: fmt.Sprintf("Adjust the chart spec using these instructions: %s. Return ONLY the updated JSON object.", instructions)},
	}
	text, err := s.llm.Chat(ctx, messages, s.chatOpts)
	if err != nil {
		return base, err
	}
	return parseChartSpec(text)
}

func parseChartSpec(text string) (domain.ChartSpec, error) {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "{"); idx >= 0 {
		text = text[idx:]
	}
	if idx := strings.LastIndex(text, "}"); idx >= 0 {
		text = text[:idx+1]
	}
	var spec domain.ChartSpec
	if err := json.Unmarshal([]byte(text), &spec); err != nil {
		return domain.ChartSpec{}, err
	}
	if spec.Type == "" {
		return domain.ChartSpec{}, fmt.Errorf("chart spec missing type")
	}
	return spec, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func schemaDescription(data *domain.QueryResult) string {
	if data == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range data.Columns {
		fmt.Fprintf(&b, "- %s (%s)\n", c.Name, c.Type)
	}
	return b.String()
}

func resultMarkdown(data *domain.QueryResult) string {
	return resultMarkdownHead(data, len(data.Rows))
}

func resultMarkdownHead(data *domain.QueryResult, limit int) string {
	if data == nil {
		return ""
	}
	var b strings.Builder
	headers := make([]string, len(data.Columns))
	for i, c := range data.Columns {
		headers[i] = c.Name
	}
	b.WriteString("| ")
	b.WriteString(strings.Join(headers, " | "))
	b.WriteString(" |\n")

	if limit > len(data.Rows) {
		limit = len(data.Rows)
	}
	for i := 0; i < limit; i++ {
		cells := make([]string, len(data.Rows[i]))
		for j, v := range data.Rows[i] {
			cells[j] = fmt.Sprint(v)
		}
		b.WriteString("| ")
		b.WriteString(strings.Join(cells, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}
