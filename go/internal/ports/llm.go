package ports

import (
	"context"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

type ChatOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
}

type LLMProvider interface {
	Chat(ctx context.Context, messages []domain.Message, opts ChatOptions) (string, error)
}
