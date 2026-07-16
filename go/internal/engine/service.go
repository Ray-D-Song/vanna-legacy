package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/ports"
)

type Service struct {
	llm        ports.LLMProvider
	vector     ports.VectorStore
	runner     ports.SQLRunner
	prompt     PromptBuilder
	chatOpts   ports.ChatOptions
	nResults   NResults
	autoTrain  bool
	visualize  bool
	allowData  bool
}

type NResults struct {
	DDL           int
	Documentation int
	SQL           int
}

type Options struct {
	LLM               ports.LLMProvider
	Vector            ports.VectorStore
	Runner            ports.SQLRunner
	Dialect           string
	Language          string
	MaxTokens         int
	Chat              ports.ChatOptions
	NResults          NResults
	AllowLLMToSeeData bool
	AutoTrain         bool
	Visualize         bool
}

func New(opts Options) *Service {
	return &Service{
		llm:    opts.LLM,
		vector: opts.Vector,
		runner: opts.Runner,
		prompt: PromptBuilder{
			Dialect:   opts.Dialect,
			MaxTokens: opts.MaxTokens,
			Language:  opts.Language,
		},
		chatOpts:  opts.Chat,
		nResults:  opts.NResults,
		allowData: opts.AllowLLMToSeeData,
		autoTrain: opts.AutoTrain,
		visualize: opts.Visualize,
	}
}

func (s *Service) Train(ctx context.Context, input domain.TrainInput) ([]string, error) {
	var ids []string
	if strings.TrimSpace(input.DDL) != "" {
		id, err := s.vector.AddDDL(ctx, input.DDL)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	if strings.TrimSpace(input.Documentation) != "" {
		id, err := s.vector.AddDocumentation(ctx, input.Documentation)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	if strings.TrimSpace(input.Question) != "" && strings.TrimSpace(input.SQL) != "" {
		id, err := s.vector.AddQuestionSQL(ctx, input.Question, input.SQL)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no training data provided")
	}
	return ids, nil
}

func (s *Service) ListTrainingData(ctx context.Context) ([]domain.TrainingItem, error) {
	return s.vector.ListTrainingData(ctx)
}

func (s *Service) RemoveTrainingData(ctx context.Context, id string) error {
	return s.vector.RemoveTrainingData(ctx, id)
}

func (s *Service) GenerateSQL(ctx context.Context, question string, allowLLMToSeeData bool) (string, error) {
	examples, err := s.vector.GetSimilarQuestionSQL(ctx, question, s.nResults.SQL)
	if err != nil {
		return "", err
	}
	ddlList, err := s.vector.GetRelatedDDL(ctx, question, s.nResults.DDL)
	if err != nil {
		return "", err
	}
	docList, err := s.vector.GetRelatedDocumentation(ctx, question, s.nResults.Documentation)
	if err != nil {
		return "", err
	}

	messages := s.prompt.BuildSQLPrompt(question, examples, ddlList, docList)
	response, err := s.llm.Chat(ctx, messages, s.chatOpts)
	if err != nil {
		return "", err
	}
	sql := ExtractSQL(response)

	if IsIntermediateSQL(sql) {
		if !allowLLMToSeeData {
			return "", fmt.Errorf("question requires database introspection; set allow_llm_to_see_data=true")
		}
		if s.runner == nil {
			return "", fmt.Errorf("database runner not configured")
		}
		intermediate := ExtractSQL(sql)
		result, err := s.runner.Run(ctx, intermediate)
		if err != nil {
			return "", fmt.Errorf("intermediate sql failed: %w", err)
		}
		docList = append(docList, fmt.Sprintf("The following is a pandas DataFrame with the results of the intermediate SQL query %s:\n%s", intermediate, resultMarkdown(result)))
		messages = s.prompt.BuildSQLPrompt(question, examples, ddlList, docList)
		response, err = s.llm.Chat(ctx, messages, s.chatOpts)
		if err != nil {
			return "", err
		}
		sql = ExtractSQL(response)
	}
	return sql, nil
}

func (s *Service) RunSQL(ctx context.Context, sql string) (*domain.QueryResult, error) {
	if s.runner == nil {
		return nil, fmt.Errorf("database runner not configured")
	}
	return s.runner.Run(ctx, sql)
}

func (s *Service) Ask(ctx context.Context, question string, opts domain.AskOptions) (*domain.AskResult, error) {
	allowData := opts.AllowLLMToSeeData || s.allowData
	sql, err := s.GenerateSQL(ctx, question, allowData)
	if err != nil {
		return nil, err
	}

	result := &domain.AskResult{Question: question, SQL: sql}
	if s.runner == nil {
		return result, nil
	}

	data, err := s.runner.Run(ctx, sql)
	if err != nil {
		return nil, err
	}
	result.Result = data

	if opts.AutoTrain || s.autoTrain {
		if !data.IsEmpty() {
			_, _ = s.vector.AddQuestionSQL(ctx, question, sql)
		}
	}

	visualize := s.visualize || opts.Visualize
	if visualize && shouldVisualize(data) {
		spec := RecommendChart(data)
		if strings.TrimSpace(opts.ChartInstructions) != "" {
			refined, err := s.RefineChartSpec(ctx, question, sql, data, spec, opts.ChartInstructions)
			if err == nil {
				spec = refined
			}
		} else {
			refined, err := s.GenerateChartSpec(ctx, question, sql, data)
			if err == nil {
				spec = refined
			}
		}
		result.Chart = &spec
	}

	if allowData {
		followups, err := s.GenerateFollowupQuestions(ctx, question, sql, data)
		if err == nil {
			result.FollowupQuestions = followups
		}
		summary, err := s.GenerateSummary(ctx, question, data)
		if err == nil {
			result.Summary = summary
		}
	}

	return result, nil
}

func (s *Service) FixSQL(ctx context.Context, question, sql, errMsg string) (string, error) {
	prompt := fmt.Sprintf("I have an error: %s\n\nHere is the SQL I tried to run: %s\n\nThis is the question I was trying to answer: %s\n\nCan you rewrite the SQL to fix the error?", errMsg, sql, question)
	return s.GenerateSQL(ctx, prompt, s.allowData)
}

func (s *Service) RewriteQuestion(ctx context.Context, lastQuestion, newQuestion string) (string, error) {
	if strings.TrimSpace(lastQuestion) == "" {
		return newQuestion, nil
	}
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: "Your goal is to combine a sequence of questions into a singular question if they are related. If the second question does not relate to the first question and is fully self-contained, return the second question. Return just the new combined question with no additional explanations. The question should theoretically be answerable with a single SQL statement."},
		{Role: domain.RoleUser, Content: "First question: " + lastQuestion + "\nSecond question: " + newQuestion},
	}
	return s.llm.Chat(ctx, messages, s.chatOpts)
}

func (s *Service) GenerateFollowupQuestions(ctx context.Context, question, sql string, data *domain.QueryResult) ([]string, error) {
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: fmt.Sprintf("You are a helpful data assistant. The user asked: '%s'\n\nSQL: %s\n\nResults preview:\n%s", question, sql, resultMarkdownHead(data, 25))},
		{Role: domain.RoleUser, Content: "Generate a list of 5 followup questions that the user might ask about this data. Respond with a list of questions, one per line. Do not answer with any explanations -- just the questions."},
	}
	text, err := s.llm.Chat(ctx, messages, s.chatOpts)
	if err != nil {
		return nil, err
	}
	return splitLines(text), nil
}

func (s *Service) GenerateSummary(ctx context.Context, question string, data *domain.QueryResult) (string, error) {
	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: fmt.Sprintf("The user asked: '%s'\n\nResults preview:\n%s", question, resultMarkdownHead(data, 25))},
		{Role: domain.RoleUser, Content: "Briefly summarize the results."},
	}
	return s.llm.Chat(ctx, messages, s.chatOpts)
}

func shouldVisualize(data *domain.QueryResult) bool {
	return data != nil && !data.IsEmpty()
}

func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line != "" {
			out = append(out, line)
		}
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}
