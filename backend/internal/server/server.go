package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"simstack/internal/orchestrator"
	"simstack/internal/types"
)

type Server struct {
	Router *http.ServeMux
	hub    *Hub
	orch   *orchestrator.Engine
}

func NewServer() *Server {
	mux := http.NewServeMux()
	hub := NewHub()
	go hub.run()

	s := &Server{
		Router: mux,
		hub:    hub,
		orch:   orchestrator.NewEngine(hub.broadcastJSON),
	}

	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// CORS for local dev: wrap mux
	s.Router = http.NewServeMux()
	s.Router.Handle("/", withCORS(mux))
	return s
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	serveWS(s.hub, w, r)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req types.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	go func() {
		// Use background context with generous timeout so it doesn't get canceled when HTTP response is sent
		// This timeout should be longer than all internal operation timeouts combined
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := s.orch.Run(ctx, req); err != nil {
			log.Printf("run error: %v", err)
			s.hub.broadcastJSON(types.WSEvent{Type: "error", Payload: map[string]any{"error": err.Error()}, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req types.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	yml, filename, err := s.orch.ExportCompose(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	_, _ = w.Write([]byte(yml))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	m := s.orch.Metrics()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m)
}

// Utility for timestamps in events
func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
