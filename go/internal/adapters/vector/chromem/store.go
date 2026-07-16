package chromem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/google/uuid"
	chromem "github.com/philippgille/chromem-go"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/ports"
)

const (
	collectionDDL           = "ddl"
	collectionDocumentation = "documentation"
	collectionSQL           = "sql"
)

type Store struct {
	db        *chromem.DB
	embedder  ports.Embedder
	nResults  nResults
	metaPath  string
	mu        sync.RWMutex
	documents map[string]domain.TrainingItem
}

type nResults struct {
	ddl           int
	documentation int
	sql           int
}

type metaFile struct {
	EmbeddingModel string                    `json:"embedding_model"`
	Dimension      int                       `json:"dimension"`
	Items          map[string]trainingRecord `json:"items"`
}

type trainingRecord struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Question string `json:"question,omitempty"`
	SQL      string `json:"sql,omitempty"`
}

func NewStore(path string, embedder ports.Embedder, nDDL, nDoc, nSQL int) (*Store, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}

	db := chromem.NewDB()
	s := &Store{
		db:       db,
		embedder: embedder,
		nResults: nResults{ddl: nDDL, documentation: nDoc, sql: nSQL},
		metaPath: filepath.Join(path, "training_meta.json"),
		documents: map[string]domain.TrainingItem{},
	}

	for _, name := range []string{collectionDDL, collectionDocumentation, collectionSQL} {
		if _, err := db.GetOrCreateCollection(name, nil, nil); err != nil {
			return nil, fmt.Errorf("create collection %s: %w", name, err)
		}
	}

	if err := s.loadMeta(); err != nil {
		return nil, err
	}
	if err := s.rebuildFromMeta(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadMeta() error {
	data, err := os.ReadFile(s.metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var meta metaFile
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	for id, rec := range meta.Items {
		s.documents[id] = domain.TrainingItem{
			ID:       id,
			Type:     rec.Type,
			Content:  rec.Content,
			Question: rec.Question,
			SQL:      rec.SQL,
		}
	}
	return nil
}

func (s *Store) saveMeta() error {
	meta := metaFile{
		EmbeddingModel: s.embedder.ModelName(),
		Dimension:      s.embedder.Dimension(),
		Items:          map[string]trainingRecord{},
	}
	for id, item := range s.documents {
		meta.Items[id] = trainingRecord{
			Type:     item.Type,
			Content:  item.Content,
			Question: item.Question,
			SQL:      item.SQL,
		}
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, data, 0o644)
}

func (s *Store) rebuildFromMeta(ctx context.Context) error {
	for id, item := range s.documents {
		if err := s.indexItem(ctx, id, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddDDL(ctx context.Context, ddl string) (string, error) {
	id := uuid.NewString()
	item := domain.TrainingItem{ID: id, Type: domain.TrainingTypeDDL, Content: ddl}
	return id, s.put(ctx, id, item, ddl)
}

func (s *Store) AddDocumentation(ctx context.Context, doc string) (string, error) {
	id := uuid.NewString()
	item := domain.TrainingItem{ID: id, Type: domain.TrainingTypeDocumentation, Content: doc}
	return id, s.put(ctx, id, item, doc)
}

func (s *Store) AddQuestionSQL(ctx context.Context, question, sql string) (string, error) {
	id := uuid.NewString()
	payload, _ := json.Marshal(map[string]string{"question": question, "sql": sql})
	item := domain.TrainingItem{ID: id, Type: domain.TrainingTypeSQL, Question: question, SQL: sql, Content: string(payload)}
	return id, s.put(ctx, id, item, string(payload))
}

func (s *Store) put(ctx context.Context, id string, item domain.TrainingItem, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.indexItem(ctx, id, item, content); err != nil {
		return err
	}
	s.documents[id] = item
	return s.saveMeta()
}

func (s *Store) indexItem(ctx context.Context, id string, item domain.TrainingItem, content ...string) error {
	text := item.Content
	if len(content) > 0 {
		text = content[0]
	}
	collectionName, err := collectionForType(item.Type)
	if err != nil {
		return err
	}
	collection, err := s.db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return err
	}

	embeddings, err := s.embedder.Embed(ctx, []string{text})
	if err != nil {
		return err
	}
	if len(embeddings) == 0 {
		return fmt.Errorf("empty embedding")
	}

	doc := chromem.Document{
		ID:        id,
		Content:   text,
		Embedding: embeddings[0],
		Metadata: map[string]string{
			"type": item.Type,
		},
	}
	return collection.AddDocuments(ctx, []chromem.Document{doc}, runtime.NumCPU())
}

func (s *Store) query(ctx context.Context, collectionName, question string, n int) ([]chromem.Result, error) {
	if n <= 0 {
		return nil, nil
	}
	collection := s.db.GetCollection(collectionName, nil)
	if collection == nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}
	embeddings, err := s.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}
	return collection.QueryEmbedding(ctx, embeddings[0], n, nil, nil)
}

func (s *Store) GetSimilarQuestionSQL(ctx context.Context, question string, n int) ([]domain.QuestionSQL, error) {
	if n <= 0 {
		n = s.nResults.sql
	}
	results, err := s.query(ctx, collectionSQL, question, n)
	if err != nil {
		return nil, err
	}
	out := make([]domain.QuestionSQL, 0, len(results))
	for _, r := range results {
		var pair struct {
			Question string `json:"question"`
			SQL      string `json:"sql"`
		}
		if err := json.Unmarshal([]byte(r.Content), &pair); err != nil {
			continue
		}
		out = append(out, domain.QuestionSQL{Question: pair.Question, SQL: pair.SQL})
	}
	return out, nil
}

func (s *Store) GetRelatedDDL(ctx context.Context, question string, n int) ([]string, error) {
	if n <= 0 {
		n = s.nResults.ddl
	}
	results, err := s.query(ctx, collectionDDL, question, n)
	if err != nil {
		return nil, err
	}
	return contents(results), nil
}

func (s *Store) GetRelatedDocumentation(ctx context.Context, question string, n int) ([]string, error) {
	if n <= 0 {
		n = s.nResults.documentation
	}
	results, err := s.query(ctx, collectionDocumentation, question, n)
	if err != nil {
		return nil, err
	}
	return contents(results), nil
}

func (s *Store) RemoveTrainingData(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.documents[id]
	if !ok {
		return fmt.Errorf("training item %s not found", id)
	}
	collectionName, err := collectionForType(item.Type)
	if err != nil {
		return err
	}
	collection := s.db.GetCollection(collectionName, nil)
	if collection == nil {
		return fmt.Errorf("collection not found: %s", collectionName)
	}
	if err := collection.Delete(ctx, nil, nil, id); err != nil {
		return err
	}
	delete(s.documents, id)
	return s.saveMeta()
}

func (s *Store) ListTrainingData(ctx context.Context) ([]domain.TrainingItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.TrainingItem, 0, len(s.documents))
	for _, item := range s.documents {
		out = append(out, item)
	}
	return out, nil
}

func collectionForType(t string) (string, error) {
	switch t {
	case domain.TrainingTypeDDL:
		return collectionDDL, nil
	case domain.TrainingTypeDocumentation:
		return collectionDocumentation, nil
	case domain.TrainingTypeSQL:
		return collectionSQL, nil
	default:
		return "", fmt.Errorf("unknown training type: %s", t)
	}
}

func contents(results []chromem.Result) []string {
	out := make([]string, len(results))
	for i, r := range results {
		out[i] = r.Content
	}
	return out
}

var _ ports.VectorStore = (*Store)(nil)
