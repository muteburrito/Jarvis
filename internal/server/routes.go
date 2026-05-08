package server

import (
	"io/fs"
	"net/http"
)

func (s *Server) registerRoutes() {
	staticSub, _ := fs.Sub(s.webFS, "static")
	fileServer := http.FileServer(http.FS(staticSub))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	s.router.Get("/", s.handleIndex)
	s.router.Post("/api/v1/chat", s.handleChat)
	s.router.Get("/api/v1/chats", s.handleListChats)
	s.router.Post("/api/v1/chats", s.handleCreateChat)
	s.router.Get("/api/v1/chats/{id}", s.handleGetChat)
	s.router.Put("/api/v1/chats/{id}", s.handleUpdateChat)
	s.router.Delete("/api/v1/chats/{id}", s.handleDeleteChat)
	s.router.Post("/api/v1/upload", s.handleUpload)
	s.router.Get("/api/v1/documents", s.handleListDocuments)
	s.router.Delete("/api/v1/documents/{id}", s.handleDeleteDocument)
	s.router.Post("/api/v1/ingest", s.handleIngestPath)
	s.router.Delete("/api/v1/documents", s.handleClearAll)
	s.router.Post("/api/v1/fetch-url", s.handleFetchURL)
	s.router.Get("/api/v1/config", s.handleAppConfig)
	s.router.Get("/api/v1/health", s.handleHealth)
	s.router.Get("/api/v1/system", s.handleSystem)
	s.router.Get("/api/v1/models", s.handleListModels)
	s.router.Get("/api/v1/projects", s.handleListProjects)
	s.router.Post("/api/v1/projects", s.handleOpenProject)
	s.router.Get("/api/v1/repo-map", s.handleRepoMap)
	s.router.Get("/api/v1/task", s.handleTaskState)
	s.router.Post("/api/v1/task/traces", s.handleAddTaskTrace)
	s.router.Post("/api/v1/task/edits", s.handleAddTaskEdit)
	s.router.Delete("/api/v1/task", s.handleClearTaskState)
	s.router.Get("/api/v1/update", s.handleUpdateStatus)
	s.router.Post("/api/v1/update/apply", s.handleUpdateApply)
}
