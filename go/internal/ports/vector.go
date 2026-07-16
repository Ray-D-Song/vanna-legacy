package ports

import (
	"context"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

type VectorStore interface {
	AddDDL(ctx context.Context, ddl string) (string, error)
	AddDocumentation(ctx context.Context, doc string) (string, error)
	AddQuestionSQL(ctx context.Context, question, sql string) (string, error)
	GetSimilarQuestionSQL(ctx context.Context, question string, n int) ([]domain.QuestionSQL, error)
	GetRelatedDDL(ctx context.Context, question string, n int) ([]string, error)
	GetRelatedDocumentation(ctx context.Context, question string, n int) ([]string, error)
	RemoveTrainingData(ctx context.Context, id string) error
	ListTrainingData(ctx context.Context) ([]domain.TrainingItem, error)
}
