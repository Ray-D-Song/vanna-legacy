package domain

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type QueryResult struct {
	Columns []Column `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

func (r *QueryResult) IsEmpty() bool {
	return r == nil || len(r.Rows) == 0
}

func (r *QueryResult) RowCount() int {
	if r == nil {
		return 0
	}
	return len(r.Rows)
}
