// Package api wires the HTTP handlers for EagleInsight's backend.
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"eagleinsight/internal/feeds"
	"eagleinsight/internal/store"
)

const (
	defaultLimit = 30
	maxLimit     = 100
	// /refresh is unauthenticated, so cap how often it can make us hit the upstream feeds.
	minRefreshInterval = 60 * time.Second
)

type Server struct {
	store       *store.Store
	mu          sync.Mutex
	lastRefresh time.Time
}

func NewServer(st *store.Store) *Server {
	return &Server{store: st}
}

// Refresh fetches every source and stores whatever's new. Safe to call from
// both the startup seed and the periodic scheduler.
func (s *Server) Refresh(ctx context.Context) (int, error) {
	s.mu.Lock()
	s.lastRefresh = time.Now()
	s.mu.Unlock()

	articles := feeds.FetchAll(ctx)
	return s.store.UpsertArticles(articles)
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/articles", s.handleArticles)
	mux.HandleFunc("GET /api/sources", s.handleSources)
	mux.HandleFunc("POST /api/refresh", s.handleRefresh)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	count, err := s.store.ArticleCount()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "articles": count})
}

func (s *Server) handleArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := defaultLimit
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimit {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "limit must be between 1 and 100"})
			return
		}
		limit = n
	}

	offset := 0
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "offset must be 0 or more"})
			return
		}
		offset = n
	}

	articles, err := s.store.QueryArticles(store.Query{
		Search: q.Get("q"),
		Source: q.Get("source"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("query articles: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if articles == nil {
		articles = []store.Article{}
	}
	writeJSON(w, http.StatusOK, articles)
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, feeds.Names())
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	tooSoon := time.Since(s.lastRefresh) < minRefreshInterval
	s.mu.Unlock()
	if tooSoon {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "refreshed recently, try again in a minute"})
		return
	}

	inserted, err := s.Refresh(r.Context())
	if err != nil {
		log.Printf("refresh: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	total, _ := s.store.ArticleCount()
	writeJSON(w, http.StatusOK, map[string]int{"inserted": inserted, "total": total})
}
