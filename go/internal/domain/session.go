package domain

import "time"

type AskState struct {
	ID                string       `json:"id"`
	Question          string       `json:"question"`
	SQL               string       `json:"sql,omitempty"`
	Result            *QueryResult `json:"result,omitempty"`
	ChartSpec         *ChartSpec   `json:"chart,omitempty"`
	FollowupQuestions []string     `json:"followup_questions,omitempty"`
	Summary           string       `json:"summary,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

type AskOptions struct {
	AllowLLMToSeeData bool   `json:"allow_llm_to_see_data"`
	AutoTrain         bool   `json:"auto_train"`
	Visualize         bool   `json:"visualize"`
	ChartInstructions string `json:"chart_instructions,omitempty"`
}

type AskResult struct {
	SessionID         string       `json:"session_id"`
	Question          string       `json:"question"`
	SQL               string       `json:"sql"`
	Result            *QueryResult `json:"data,omitempty"`
	Chart             *ChartSpec   `json:"chart,omitempty"`
	FollowupQuestions []string     `json:"followup_questions,omitempty"`
	Summary           string       `json:"summary,omitempty"`
}

type GenerateSQLOptions struct {
	AllowLLMToSeeData bool
}
