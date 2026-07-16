package ports

import (
	"context"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

type SQLRunner interface {
	Run(ctx context.Context, sql string) (*domain.QueryResult, error)
	Dialect() string
	Close() error
}
