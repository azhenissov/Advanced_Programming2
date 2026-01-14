package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/azhenissov/Advanced_Programming2/internal/model"
	"github.com/azhenissov/Advanced_Programming2/internal/store"
)

const (
	RateLimitWindow = 10 * time.Second
	MaxRequests     = 5
)

type Server struct {
	store          *store.Store[string, string]
	clientState    *store.Store[string, model.ClientState]
	requests       atomic.Int64
	startTime      time.Time
	mux            *http.ServeMux
	resetTicker    *time.Ticker
	stopChan       chan struct{}
	mu             sync.Mutex
	blockedClients map[string]bool
}

func New() *Server {
	s := &Server{
		store:          store.New[string, string](),
		clientState:    store.New[string, model.ClientState](),
		startTime:      time.Now(),
		mux:            http.NewServeMux(),
		resetTicker:    time.NewTicker(RateLimitWindow),
		stopChan:       make(chan struct{}),
		blockedClients: make(map[string]bool),
	}
	s.registerRoutes()
	go s.resetLoop()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /data", s.handlePostData)
	s.mux.HandleFunc("GET /data", s.handleGetAllData)
	s.mux.HandleFunc("GET /data/{key}", s.handleGetData)
	s.mux.HandleFunc("GET /ping", s.handlePing)
	s.mux.HandleFunc("DELETE /data/{key}", s.handleDeleteData)
	s.mux.HandleFunc("GET /stats", s.handleGetStats)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests.Add(1)
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) resetLoop() {
	for {
		select {
		case <-s.resetTicker.C:
			s.resetClientCounters()
		case <-s.stopChan:
			return
		}
	}
}

func (s *Server) resetClientCounters() {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients := s.clientState.Keys()
	now := time.Now()

	for _, clientID := range clients {
		if state, ok := s.clientState.Get(clientID); ok {
			if now.Sub(state.LastReset) >= RateLimitWindow {
				state.RequestCount = 0
				state.LastReset = now
				state.IsBlocked = false
				s.clientState.Set(clientID, state)
				delete(s.blockedClients, clientID)
			}
		}
	}
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

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client")
	if clientID == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"status": "client parameter required"}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if blocked, exists := s.blockedClients[clientID]; exists && blocked {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"status": "rate limit exceeded"})
		return
	}

	state, exists := s.clientState.Get(clientID)
	if !exists {
		state = model.ClientState{
			RequestCount: 0,
			LastReset:    time.Now(),
			IsBlocked:    false,
		}
	}

	if time.Since(state.LastReset) >= RateLimitWindow {
		state.RequestCount = 0
		state.LastReset = time.Now()
		state.IsBlocked = false
		delete(s.blockedClients, clientID)
	}

	state.RequestCount++

	if state.RequestCount > MaxRequests {
		state.IsBlocked = true
		s.blockedClients[clientID] = true
		s.clientState.Set(clientID, state)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"status": "rate limit exceeded"})
		return
	}

	s.clientState.Set(clientID, state)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !s.store.Delete(key) {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type StatsResponse struct {
	Clients        int   `json:"clients"`
	TotalRequests  int64 `json:"total_requests"`
	BlockedClients int   `json:"blocked_clients"`
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	blockedCount := len(s.blockedClients)
	s.mu.Unlock()

	clientCount := s.clientState.Count()

	stats := StatsResponse{
		Clients:        clientCount,
		TotalRequests:  s.requests.Load(),
		BlockedClients: blockedCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) GetRequestsPtr() *atomic.Int64 {
	return &s.requests
}

func (s *Server) GetKeyCount() int {
	return s.store.Count()
}

func (s *Server) Stop() {
	s.resetTicker.Stop()
	close(s.stopChan)
}
