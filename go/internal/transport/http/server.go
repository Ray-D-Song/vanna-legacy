package http

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/engine"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/session"
)

//go:embed web/*
var webFS embed.FS

type Server struct {
	engine  *engine.Service
	session *session.Store
	router  chi.Router
}

func NewServer(svc *engine.Service, sessions *session.Store) *Server {
	s := &Server{engine: svc, session: sessions}
	s.router = s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/ask", s.handleAsk)
		api.Post("/generate_sql", s.handleGenerateSQL)
		api.Post("/run_sql", s.handleRunSQL)
		api.Post("/fix_sql", s.handleFixSQL)
		api.Post("/update_sql", s.handleUpdateSQL)
		api.Post("/chart", s.handleChart)
		api.Get("/followup_questions", s.handleFollowups)
		api.Get("/summary", s.handleSummary)
		api.Post("/rewrite_question", s.handleRewriteQuestion)
		api.Post("/train", s.handleTrain)
		api.Get("/training_data", s.handleListTraining)
		api.Delete("/training_data/{id}", s.handleDeleteTraining)
		api.Get("/sessions", s.handleListSessions)
		api.Get("/sessions/{id}", s.handleGetSession)
	})

	webRoot, _ := fs.Sub(webFS, "web")
	fileServer := http.FileServer(http.FS(webRoot))
	r.Handle("/*", fileServer)
	return r
}

type askRequest struct {
	Question              string `json:"question"`
	AllowLLMToSeeData     *bool  `json:"allow_llm_to_see_data"`
	AutoTrain             *bool  `json:"auto_train"`
	Visualize             *bool  `json:"visualize"`
	ChartInstructions     string `json:"chart_instructions"`
}

type sessionRequest struct {
	SessionID string `json:"session_id"`
}

type generateSQLRequest struct {
	Question          string `json:"question"`
	AllowLLMToSeeData *bool  `json:"allow_llm_to_see_data"`
}

type runSQLRequest struct {
	SessionID string `json:"session_id"`
	SQL       string `json:"sql,omitempty"`
}

type fixSQLRequest struct {
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

type updateSQLRequest struct {
	SessionID string `json:"session_id"`
	SQL       string `json:"sql"`
}

type chartRequest struct {
	SessionID         string `json:"session_id"`
	ChartInstructions string `json:"chart_instructions"`
}

type rewriteRequest struct {
	LastQuestion string `json:"last_question"`
	NewQuestion  string `json:"new_question"`
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	opts := domain.AskOptions{
		ChartInstructions: req.ChartInstructions,
	}
	if req.AllowLLMToSeeData != nil {
		opts.AllowLLMToSeeData = *req.AllowLLMToSeeData
	}
	if req.AutoTrain != nil {
		opts.AutoTrain = *req.AutoTrain
	}
	if req.Visualize != nil {
		opts.Visualize = *req.Visualize
	} else {
		opts.Visualize = true
	}

	result, err := s.engine.Ask(r.Context(), req.Question, opts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	state := s.session.Create(domain.AskState{
		Question:          result.Question,
		SQL:               result.SQL,
		Result:            result.Result,
		ChartSpec:         result.Chart,
		FollowupQuestions: result.FollowupQuestions,
		Summary:           result.Summary,
	})
	result.SessionID = state.ID
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGenerateSQL(w http.ResponseWriter, r *http.Request) {
	var req generateSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	allow := false
	if req.AllowLLMToSeeData != nil {
		allow = *req.AllowLLMToSeeData
	}
	sql, err := s.engine.GenerateSQL(r.Context(), req.Question, allow)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state := s.session.Create(domain.AskState{Question: req.Question, SQL: sql})
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": state.ID,
		"sql":        sql,
		"valid":      engine.IsSQLValid(sql),
	})
}

func (s *Server) handleRunSQL(w http.ResponseWriter, r *http.Request) {
	var req runSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sessionID := firstNonEmpty(req.SessionID, r.Header.Get("X-Session-ID"))
	state, ok := s.session.Get(sessionID)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	sql := state.SQL
	if strings.TrimSpace(req.SQL) != "" {
		sql = req.SQL
	}
	result, err := s.engine.RunSQL(r.Context(), sql)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, _ := s.session.Update(sessionID, func(st *domain.AskState) {
		st.SQL = sql
		st.Result = result
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": updated.ID,
		"sql":        updated.SQL,
		"data":       updated.Result,
	})
}

func (s *Server) handleFixSQL(w http.ResponseWriter, r *http.Request) {
	var req fixSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, ok := s.session.Get(firstNonEmpty(req.SessionID, r.Header.Get("X-Session-ID")))
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	sql, err := s.engine.FixSQL(r.Context(), state.Question, state.SQL, req.Error)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, _ := s.session.Update(state.ID, func(st *domain.AskState) {
		st.SQL = sql
	})
	writeJSON(w, http.StatusOK, map[string]any{"session_id": updated.ID, "sql": sql})
}

func (s *Server) handleUpdateSQL(w http.ResponseWriter, r *http.Request) {
	var req updateSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, ok := s.session.Update(firstNonEmpty(req.SessionID, r.Header.Get("X-Session-ID")), func(st *domain.AskState) {
		st.SQL = req.SQL
	})
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": updated.ID, "sql": updated.SQL})
}

func (s *Server) handleChart(w http.ResponseWriter, r *http.Request) {
	var req chartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, ok := s.session.Get(firstNonEmpty(req.SessionID, r.Header.Get("X-Session-ID")))
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if state.Result == nil {
		writeError(w, http.StatusBadRequest, "no query result in session")
		return
	}
	spec := engine.RecommendChart(state.Result)
	var err error
	if strings.TrimSpace(req.ChartInstructions) != "" {
		spec, err = s.engine.RefineChartSpec(r.Context(), state.Question, state.SQL, state.Result, spec, req.ChartInstructions)
	} else {
		spec, err = s.engine.GenerateChartSpec(r.Context(), state.Question, state.SQL, state.Result)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, _ := s.session.Update(state.ID, func(st *domain.AskState) {
		st.ChartSpec = &spec
	})
	writeJSON(w, http.StatusOK, map[string]any{"session_id": updated.ID, "chart": spec})
}

func (s *Server) handleFollowups(w http.ResponseWriter, r *http.Request) {
	state, ok := s.loadSessionFromRequest(r)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if state.Result == nil {
		writeError(w, http.StatusBadRequest, "no query result in session")
		return
	}
	questions, err := s.engine.GenerateFollowupQuestions(r.Context(), state.Question, state.SQL, state.Result)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, _ := s.session.Update(state.ID, func(st *domain.AskState) {
		st.FollowupQuestions = questions
	})
	writeJSON(w, http.StatusOK, map[string]any{"session_id": updated.ID, "questions": questions})
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	state, ok := s.loadSessionFromRequest(r)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	if state.Result == nil {
		writeError(w, http.StatusBadRequest, "no query result in session")
		return
	}
	summary, err := s.engine.GenerateSummary(r.Context(), state.Question, state.Result)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, _ := s.session.Update(state.ID, func(st *domain.AskState) {
		st.Summary = summary
	})
	writeJSON(w, http.StatusOK, map[string]any{"session_id": updated.ID, "summary": summary})
}

func (s *Server) handleRewriteQuestion(w http.ResponseWriter, r *http.Request) {
	var req rewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	question, err := s.engine.RewriteQuestion(r.Context(), req.LastQuestion, req.NewQuestion)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"question": question})
}

func (s *Server) handleTrain(w http.ResponseWriter, r *http.Request) {
	var req domain.TrainInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ids, err := s.engine.Train(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
}

func (s *Server) handleListTraining(w http.ResponseWriter, r *http.Request) {
	items, err := s.engine.ListTrainingData(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleDeleteTraining(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.engine.RemoveTrainingData(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"questions": s.session.ListQuestions()})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	state, ok := s.session.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) loadSessionFromRequest(r *http.Request) (*domain.AskState, bool) {
	id := firstNonEmpty(r.URL.Query().Get("session_id"), r.Header.Get("X-Session-ID"))
	return s.session.Get(id)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func Shutdown(ctx context.Context, srv *http.Server) error {
	return srv.Shutdown(ctx)
}
