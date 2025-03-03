package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/njhale/maskfs/pkg/index"
	"github.com/njhale/maskfs/pkg/logger"
	"github.com/njhale/maskfs/pkg/mask"
)

// Config represents the server configuration
type Config struct {
	Port string `usage:"Port to listen on" default:"9888"`
	Mask string `usage:"Path mask to apply to the server" default:"**/maskfs/\n**/*.go"`
}

// Server represents a secure HTTP file server with glob-based filtering
type Server struct {
	mask   index.Mask
	logger logger.Logger
}

// New creates a new FileServer instance
func New(cfg Config) (*Server, error) {
	pathMask, err := mask.NewGlobMask(cfg.Mask)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path mask: %w", err)
	}

	return &Server{
		mask:   pathMask,
		logger: logger.New("server"),
	}, nil
}

// Run starts the file server
func Run(ctx context.Context, cfg Config) error {
	server, err := New(cfg)
	if err != nil {
		return err
	}
	server.logger.Debugf("Server created with mask: %#v", server.mask)

	// Set up the default HTTP muxer
	mux := http.NewServeMux()

	// Register root handler that always returns 200 OK
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.logger.Debugf("Root handler called: %s", r.URL.Path)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Register the file server under /files/
	mux.Handle("/files/", http.StripPrefix("/files/", server))

	// Create and start the HTTP server
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	errCh := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		server.logger.Debugf("Starting server on port: %s", cfg.Port)
		errCh <- httpServer.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		server.logger.Debugf("Context canceled, shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != http.ErrServerClosed {
			server.logger.Errorf("Server error: %v", err)
			return err
		}
		return nil
	}
}

// ServeHTTP handles file requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Debugf("Handling request %s: %s", r.Method, r.URL.Path)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clean and normalize the path
	fsPath, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if fsPath = filepath.Clean(fsPath); fsPath == "." {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	s.logger.Debugf("Serving path: %q", fsPath)

	// Get entry info
	dirFS := os.DirFS("/")
	entry, err := index.GetEntry(dirFS, fsPath)
	if err != nil {
		s.logger.Errorf("Error getting entry\n", err)
		http.NotFound(w, r)
		return
	}

	s.logger.Debugf("Got entry from filesystem: %#v", entry)

	if s.mask.Masked(entry) {
		// The client-requested entry is masked, return a 404.
		s.logger.Debugf("Entry %q is masked, returning 404\n\t%q", entry.FSPath, entry.LinkPath)
		http.NotFound(w, r)
		return
	}

	if entry.IsDir {
		// The client-requested entry is an unmasked directory, render a masked index of its immediate children.
		masked, err := index.GetEntries(dirFS, entry.FSPath, s.mask)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Sort by name to ensure the entry order in the rendered HTML is consistent.
		sort.Slice(masked, func(i, j int) bool {
			return masked[i].Name < masked[j].Name
		})

		if err := masked.WriteHTML(w, entry, masked); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		return
	}

	// The entry is an unmasked file, serve its contents using ServeFileFS.
	http.ServeFileFS(w, r, dirFS, entry.FSPath)
}
