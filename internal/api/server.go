// Package api provides the HTTP server and handlers for Mnemosyne.
package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/ali01/mnemosyne/internal/indexer"
	"github.com/ali01/mnemosyne/internal/store"
)

// sseEvent carries typed event data to SSE clients.
type sseEvent struct {
	Type     string `json:"type"`               // "graph-updated" or "graphs-changed"
	GraphIDs []int  `json:"graphIds,omitempty"`
}

// Server is the HTTP server for Mnemosyne.
type Server struct {
	store   *store.Store
	indexer *indexer.IndexManager
	mux     *http.ServeMux
	port    int

	sseClients   map[chan sseEvent]struct{}
	sseClientsMu sync.Mutex
}

// NewServer creates a new HTTP server.
func NewServer(s *store.Store, idx *indexer.IndexManager, staticFS fs.FS, port int) *Server {
	srv := &Server{
		store:      s,
		indexer:    idx,
		mux:        http.NewServeMux(),
		port:       port,
		sseClients: make(map[chan sseEvent]struct{}),
	}

	// API routes
	srv.mux.HandleFunc("GET /api/v1/events", srv.handleSSE)
	srv.mux.HandleFunc("GET /api/v1/health", srv.handleHealth)

	// Graph listing and data
	srv.mux.HandleFunc("GET /api/v1/graphs", srv.handleListGraphs)
	srv.mux.HandleFunc("GET /api/v1/graphs/{id}", srv.handleGetGraphData)
	srv.mux.HandleFunc("GET /api/v1/graphs/{id}/search", srv.handleSearchInGraph)

	// Graph-scoped positions
	srv.mux.HandleFunc("PUT /api/v1/graphs/{id}/positions", srv.handleUpdateGraphPositions)
	srv.mux.HandleFunc("PUT /api/v1/graphs/{id}/positions/{nodeId}", srv.handleUpdateGraphPosition)

	// Node content (not graph-scoped)
	srv.mux.HandleFunc("GET /api/v1/nodes/{id}", srv.handleGetNode)
	srv.mux.HandleFunc("GET /api/v1/nodes/{id}/content", srv.handleGetNodeContent)

	// Reindex
	srv.mux.HandleFunc("POST /api/v1/reindex", srv.handleReindex)

	// Static files with SPA fallback
	if staticFS != nil {
		srv.mux.Handle("/", spaHandler(staticFS))
	}

	return srv
}

// Handler returns the http.Handler.
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

// NotifyChange broadcasts a graph-updated event to all SSE clients.
func (s *Server) NotifyChange(graphIDs []int) {
	s.broadcast(sseEvent{Type: "graph-updated", GraphIDs: graphIDs})
}

// NotifyGraphsChanged broadcasts a graphs-changed event to all SSE clients.
func (s *Server) NotifyGraphsChanged() {
	s.broadcast(sseEvent{Type: "graphs-changed"})
}

func (s *Server) broadcast(evt sseEvent) {
	s.sseClientsMu.Lock()
	defer s.sseClientsMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- evt:
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

	ch := make(chan sseEvent, 1)
	s.sseClientsMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseClientsMu.Unlock()

	defer func() {
		s.sseClientsMu.Lock()
		delete(s.sseClients, ch)
		s.sseClientsMu.Unlock()
	}()

	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-ch:
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
			flusher.Flush()
		}
	}
}

// spaHandler serves static files and falls back to index.html for unmatched routes.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		if _, err := fs.Stat(staticFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

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
