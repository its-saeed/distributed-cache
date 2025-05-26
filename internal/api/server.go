package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/its-saeed/distributed-cache/internal/cache"
)

// Server represents the HTTP API server
type Server struct {
	cacheManager *cache.Manager
	httpServer   *http.Server
	shutdown     chan struct{}
}

// NewServer initializes a new API server
func NewServer(addr string, cacheManager *cache.Manager) *Server {
	s := &Server{
		cacheManager: cacheManager,
		shutdown:     make(chan struct{}),
	}

	// Define HTTP routes
	router := http.NewServeMux()
	router.HandleFunc("/", s.handleRoot)
	router.HandleFunc("/get", s.handleGet)
	router.HandleFunc("/set", s.handleSet)
	router.HandleFunc("/del", s.handleDelete)

	// Configure the HTTP server
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return s
}

func (s *Server) Run(ctx context.Context) {
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	<-ctx.Done() // Wait for shutdown signal
	log.Println("Shutting down API server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("API server shutdown error: %v", err)
	}

	log.Println("API server shut down cleanly.")
}

// Start runs the HTTP server
func (s *Server) Start() error {
	go s.gracefulShutdown()
	return s.httpServer.ListenAndServe()
}

// Stop initiates a graceful shutdown
func (s *Server) Shutdown() {
	close(s.shutdown)
}

// gracefulShutdown handles clean server termination
func (s *Server) gracefulShutdown() {
	<-s.shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.httpServer.Shutdown(ctx)
}

// handleRoot serves the root endpoint
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "Endpoint not found")
		return
	}
	writeResponse(w, http.StatusOK, map[string]string{
		"message": "Distributed Cache API",
		"version": "1.0",
	})
}

// handleGet handles cache retrieval
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "Missing 'key' parameter")
		return
	}

	node, err := s.cacheManager.GetNodeForKey(key)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	value, found := s.cacheManager.GetKeyFromNode(node.String(), key)
	if !found {
		writeError(w, http.StatusNotFound, "Key not found")
		return
	}

	writeResponse(w, http.StatusOK, map[string]interface{}{
		"key":   key,
		"value": string(value),
		"node":  node.String(),
	})
}

// handleSet handles key-value storage
func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	ttl_str := r.URL.Query().Get("ttl")
	if key == "" || value == "" {
		writeError(w, http.StatusBadRequest, "Both 'key' and 'value' parameters are required")
		return
	}

	ttl, err := strconv.Atoi(ttl_str)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ttl is not a number")
		return
	}

	node, err := s.cacheManager.GetNodeForKey(key)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	err = s.cacheManager.SetKeyOnNode(node.String(), key, []byte(value), ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Value set successfully",
		"key":     key,
		"node":    node.String(),
	})
}

// handleSet handles key-value storage
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "`key` parameter is required")
		return
	}

	node, err := s.cacheManager.GetNodeForKey(key)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	err = s.cacheManager.DeleteKeyOnNode(node.String(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Value deleted successfully",
		"key":     key,
		"node":    node.String(),
	})
}

// writeError sends an error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeResponse sends a success response
func writeResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
