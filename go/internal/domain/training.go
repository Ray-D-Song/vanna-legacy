package domain

const (
	TrainingTypeDDL           = "ddl"
	TrainingTypeDocumentation = "documentation"
	TrainingTypeSQL           = "sql"
)

type QuestionSQL struct {
	Question string `json:"question"`
	SQL      string `json:"sql"`
}

type TrainingItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Question string `json:"question,omitempty"`
	SQL      string `json:"sql,omitempty"`
}

type TrainInput struct {
	DDL           string `json:"ddl,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Question      string `json:"question,omitempty"`
	SQL           string `json:"sql,omitempty"`
}
