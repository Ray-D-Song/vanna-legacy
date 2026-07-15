package ports

import "context"

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}
