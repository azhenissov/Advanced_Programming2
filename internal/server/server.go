package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/azhenissov/Advanced_Programming2/internal/model"
	"github.com/azhenissov/Advanced_Programming2/internal/store"
)

type Server struct {
	store     *store.Store[string, string]
	requests  atomic.Int64
	startTime time.Time
	mux       *http.ServeMux
}

func New() *Server {
	s := &Server{
		store:     store.New[string, string](),
		startTime: time.Now(),
		mux:       http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /data", s.handlePostData)
	s.mux.HandleFunc("GET /data", s.handleGetAllData)
	s.mux.HandleFunc("GET /data/{key}", s.handleGetData)
	s.mux.HandleFunc("DELETE /data/{key}", s.handleDeleteData)
	s.mux.HandleFunc("GET /stats", s.handleGetStats)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) handlePostData(w http.ResponseWriter, r *http.Request) {
	var req model.DataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Key == "" || req.Value == "" {
		http.Error(w, "Key and value are required", http.StatusBadRequest)
		return
	}

	s.store.Set(req.Key, req.Value)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (s *Server) handleGetAllData(w http.ResponseWriter, r *http.Request) {
	data := s.store.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleGetData(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, ok := s.store.Get(key)
	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{key: value})
}

func (s *Server) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !s.store.Delete(key) {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats := model.Stats{
		Requests:      s.requests.Load(),
		Keys:          s.store.Count(),
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) GetRequestsPtr() *atomic.Int64 {
	return &s.requests
}

func (s *Server) GetKeyCount() int {
	return s.store.Count()
}
