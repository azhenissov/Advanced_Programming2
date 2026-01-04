package server

import (
	"net/http"
	"encoding/json"
	"sync/atomic"
	"strings"
	"time"

	"github.com/azhenissov/Advanced_Programming2/internal/model"
	"github.com/azhenissov/Advanced_Programming2/internal/store"
)

type Server struct {
	store *store.Store[string, string]
	startTime time.Time
	requestCount uint64
}

func NewServer(store *store.Store[string, string]) *Server {
	return &Server{
		store: store,
		startTime: time.Now(),
	}
}

func (s *Server) RequestCount() uint64 {
	return atomic.LoadUint64(&s.requestCount)
}

func (s *Server) KeyCount() int {
	return s.store.Len()
}

func (s *Server) increment() {
	atomic.AddUint64(&s.requestCount, 1)
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/data", s.handleData)
	mux.HandleFunc("/data/", s.handleDataKey)
	mux.HandleFunc("/stats", s.handleStats)
}

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	s.increment()

	switch r.Method {
	case http.MethodPost:
		var req model.DataRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || 
			req.Key == "" || req.Value == "" {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		s.store.Set(req.Key, req.Value)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	
	case http.MethodGet:
		json.NewEncoder(w).Encode(s.store.Snapshot())

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDataKey(w http.ResponseWriter, r *http.Request) {
	s.increment()

	key := strings.TrimPrefix(r.URL.Path, "/data/")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		val, ok := s.store.Get(key)
		if !ok {
			http.Error(w, "Key not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{key: val})

	case http.MethodDelete:
		if !s.store.Delete(key) {
			http.Error(w, "Key not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.increment()

	stats := model.Stats{
		Requests: int64(s.RequestCount()),
		Keys:     s.KeyCount(),
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}
	json.NewEncoder(w).Encode(stats)
}
