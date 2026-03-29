// Package api provides the HTTP server and handlers for Mnemosyne.
package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
)

// Server is the HTTP server for Mnemosyne.
type Server struct {
	store   *store.Store
	indexer *indexer.Indexer
	mux     *http.ServeMux
	port    int

	sseClients   map[chan struct{}]struct{}
	sseClientsMu sync.Mutex
}

// NewServer creates a new HTTP server.
func NewServer(s *store.Store, idx *indexer.Indexer, staticFS fs.FS, port int) *Server {
	srv := &Server{
		store:      s,
		indexer:    idx,
		mux:        http.NewServeMux(),
		port:       port,
		sseClients: make(map[chan struct{}]struct{}),
	}

	// API routes
	srv.mux.HandleFunc("GET /api/v1/events", srv.handleSSE)
	srv.mux.HandleFunc("GET /api/v1/health", srv.handleHealth)
	srv.mux.HandleFunc("GET /api/v1/graph", srv.handleGetGraph)
	srv.mux.HandleFunc("GET /api/v1/nodes/search", srv.handleSearchNodes)
	srv.mux.HandleFunc("GET /api/v1/nodes/{id}", srv.handleGetNode)
	srv.mux.HandleFunc("GET /api/v1/nodes/{id}/content", srv.handleGetNodeContent)
	srv.mux.HandleFunc("PUT /api/v1/nodes/{id}/position", srv.handleUpdatePosition)
	srv.mux.HandleFunc("PUT /api/v1/nodes/positions", srv.handleUpdatePositions)
	srv.mux.HandleFunc("POST /api/v1/vault/reindex", srv.handleReindex)

	// Static files with SPA fallback
	if staticFS != nil {
		srv.mux.Handle("/", spaHandler(staticFS))
	}

	return srv
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Listening on http://localhost:%d", s.port)
	return http.ListenAndServe(addr, corsMiddleware(s.mux))
}

// Handler returns the http.Handler for testing.
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

// NotifyChange broadcasts a graph-changed event to all SSE clients.
func (s *Server) NotifyChange() {
	s.sseClientsMu.Lock()
	defer s.sseClientsMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// handleSSE streams server-sent events to the client.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 1)
	s.sseClientsMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseClientsMu.Unlock()

	defer func() {
		s.sseClientsMu.Lock()
		delete(s.sseClients, ch)
		s.sseClientsMu.Unlock()
	}()

	// Send initial connected event
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "event: graph-updated\ndata: reload\n\n")
			flusher.Flush()
		}
	}
}

// spaHandler serves static files and falls back to index.html for unmatched routes.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Check if file exists
		if _, err := fs.Stat(staticFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-API, non-file routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
