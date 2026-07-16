package domain

const (
	ChartTypeBar     = "bar"
	ChartTypeLine    = "line"
	ChartTypeScatter = "scatter"
	ChartTypePie     = "pie"
	ChartTypeMetric  = "metric"
	ChartTypeTable   = "table"
)

type FieldRef struct {
	Field string `json:"field"`
	Label string `json:"label,omitempty"`
}

type SortSpec struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

type ChartSpec struct {
	Type        string     `json:"type"`
	Title       string     `json:"title,omitempty"`
	X           *FieldRef  `json:"x,omitempty"`
	Y           *FieldRef  `json:"y,omitempty"`
	Series      []FieldRef `json:"series,omitempty"`
	GroupBy     string     `json:"group_by,omitempty"`
	Sort        *SortSpec  `json:"sort,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Stacked     bool       `json:"stacked,omitempty"`
	ValueField  string     `json:"value_field,omitempty"`
	ValueLabel  string     `json:"value_label,omitempty"`
}
