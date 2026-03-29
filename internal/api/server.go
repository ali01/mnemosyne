// Package api provides the HTTP server and handlers for Mnemosyne.
package api

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
)

// Server is the HTTP server for Mnemosyne.
type Server struct {
	store   *store.Store
	indexer *indexer.Indexer
	mux     *http.ServeMux
	port    int
}

// NewServer creates a new HTTP server.
func NewServer(s *store.Store, idx *indexer.Indexer, staticFS fs.FS, port int) *Server {
	srv := &Server{
		store:   s,
		indexer: idx,
		mux:     http.NewServeMux(),
		port:    port,
	}

	// API routes
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
