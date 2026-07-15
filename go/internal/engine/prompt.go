package engine

import (
	"fmt"
	"strings"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

type PromptBuilder struct {
	Dialect             string
	MaxTokens           int
	StaticDocumentation string
	Language            string
}

func (p *PromptBuilder) BuildSQLPrompt(question string, examples []domain.QuestionSQL, ddlList, docList []string) []domain.Message {
	initial := fmt.Sprintf("You are a %s expert. Please help to generate a SQL query to answer the question. Your response should ONLY be based on the given context and follow the response guidelines and format instructions.\n", p.Dialect)
	initial = addDDL(initial, ddlList, p.MaxTokens)
	docs := append([]string{}, docList...)
	if strings.TrimSpace(p.StaticDocumentation) != "" {
		docs = append(docs, p.StaticDocumentation)
	}
	initial = addDocumentation(initial, docs, p.MaxTokens)
	initial += "===Response Guidelines \n" +
		"1. If the provided context is sufficient, please generate a valid SQL query without any explanations for the question. \n" +
		"2. If the provided context is almost sufficient but requires knowledge of a specific string in a particular column, please generate an intermediate SQL query to find the distinct strings in that column. Prepend the query with a comment saying intermediate_sql \n" +
		"3. If the provided context is insufficient, please explain why it can't be generated. \n" +
		"4. Please use the most relevant table(s). \n" +
		"5. If the question has been asked and answered before, please repeat the answer exactly as it was given before. \n" +
		fmt.Sprintf("6. Ensure that the output SQL is %s-compliant and executable, and free of syntax errors. \n", p.Dialect)

	messages := []domain.Message{{Role: domain.RoleSystem, Content: initial}}
	for _, ex := range examples {
		if ex.Question == "" || ex.SQL == "" {
			continue
		}
		messages = append(messages,
			domain.Message{Role: domain.RoleUser, Content: ex.Question},
			domain.Message{Role: domain.RoleAssistant, Content: ex.SQL},
		)
	}
	messages = append(messages, domain.Message{Role: domain.RoleUser, Content: question + p.responseLanguageSuffix()})
	return messages
}

func (p *PromptBuilder) responseLanguageSuffix() string {
	if strings.TrimSpace(p.Language) == "" {
		return ""
	}
	return "\n\nRespond in the " + p.Language + " language."
}

func addDDL(prompt string, ddlList []string, maxTokens int) string {
	if len(ddlList) == 0 {
		return prompt
	}
	prompt += "\n===Tables \n"
	for _, ddl := range ddlList {
		if approxTokens(prompt)+approxTokens(ddl) < maxTokens {
			prompt += ddl + "\n\n"
		}
	}
	return prompt
}

func addDocumentation(prompt string, docList []string, maxTokens int) string {
	if len(docList) == 0 {
		return prompt
	}
	prompt += "\n===Additional Context \n\n"
	for _, doc := range docList {
		if approxTokens(prompt)+approxTokens(doc) < maxTokens {
			prompt += doc + "\n\n"
		}
	}
	return prompt
}

func approxTokens(s string) int {
	return len(s) / 4
}
