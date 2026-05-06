package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"go-chatbot/internal/config"
	"go-chatbot/internal/document"
	"go-chatbot/internal/rag"
	"go-chatbot/internal/updater"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/websearch"
	"go-chatbot/internal/workbench"
)

type Server struct {
	router     chi.Router
	cfg        *config.Config
	processor  *document.Processor
	chain      *rag.Chain
	store      *vectorstore.Store
	researcher *websearch.Researcher
	tasks      *workbench.TaskStore
	chats      *workbench.ChatStore
	updater    *updater.Updater
	webFS      fs.FS
}

type Dependencies struct {
	Config    *config.Config
	Processor *document.Processor
	Chain     *rag.Chain
	Store     *vectorstore.Store
	Research  *websearch.Researcher
	Tasks     *workbench.TaskStore
	Chats     *workbench.ChatStore
	Updater   *updater.Updater
	WebFS     fs.FS
}

func New(deps Dependencies) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		cfg:        deps.Config,
		processor:  deps.Processor,
		chain:      deps.Chain,
		store:      deps.Store,
		researcher: deps.Research,
		tasks:      deps.Tasks,
		chats:      deps.Chats,
		updater:    deps.Updater,
		webFS:      deps.WebFS,
	}

	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}))

	s.registerRoutes()
	return s
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(s.webFS, "templates/index.html")
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("server starting", "port", s.cfg.Port, "url", fmt.Sprintf("http://localhost:%d", s.cfg.Port))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
