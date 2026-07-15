package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/ports"
)

type Runner struct {
	db *sql.DB
}

func NewRunner(dsn string) (*Runner, error) {
	if strings.TrimSpace(dsn) == "" {
		dsn = "vanna.duckdb"
	}
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Runner{db: db}, nil
}

func (r *Runner) Run(ctx context.Context, query string) (*domain.QueryResult, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	columnTypes, _ := rows.ColumnTypes()

	result := &domain.QueryResult{
		Columns: make([]domain.Column, len(cols)),
	}
	for i, name := range cols {
		colType := "unknown"
		if i < len(columnTypes) && columnTypes[i] != nil {
			colType = mapColumnType(columnTypes[i].DatabaseTypeName())
		}
		result.Columns[i] = domain.Column{Name: name, Type: colType}
	}

	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]any, len(cols))
		for i, v := range values {
			row[i] = normalizeValue(v)
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func (r *Runner) Dialect() string {
	return "DuckDB"
}

func (r *Runner) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func mapColumnType(dbType string) string {
	t := strings.ToLower(dbType)
	switch {
	case strings.Contains(t, "int"), strings.Contains(t, "float"), strings.Contains(t, "double"), strings.Contains(t, "decimal"), strings.Contains(t, "numeric"), strings.Contains(t, "hugeint"):
		return "number"
	case strings.Contains(t, "time"), strings.Contains(t, "date"), strings.Contains(t, "timestamp"):
		return "datetime"
	case strings.Contains(t, "bool"):
		return "boolean"
	default:
		return "string"
	}
}

func normalizeValue(v any) any {
	switch val := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return val
	}
}

var _ ports.SQLRunner = (*Runner)(nil)

func Open(dsn string) (ports.SQLRunner, error) {
	r, err := NewRunner(dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdb: %w", err)
	}
	return r, nil
}
