package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	state, err := workbench.LoadProjectState(s.cfg.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, workbench.ProjectState{})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load projects")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleOpenProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	info, err := os.Stat(req.Path)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a valid directory")
		return
	}

	state, project, err := workbench.UpsertProject(s.cfg.DataDir, req.Path, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save project")
		return
	}

	repoMap, err := workbench.ScanRepository(project.Path)
	if err != nil {
		slog.Warn("failed to build project workspace map", "path", project.Path, "error", err)
	} else if err := workbench.SaveRepoMap(s.cfg.DataDir, repoMap); err != nil {
		slog.Warn("failed to save project workspace map", "path", project.Path, "error", err)
	}

	s.recordTaskTrace("project_open", "Opened project", map[string]string{
		"name":             project.Name,
		"path":             project.Path,
		"vector_store_dir": project.VectorStoreDir,
	})

	documentCount := 0
	if store, _, err := s.projectStore(project); err == nil && store != nil {
		documentCount = store.DocumentCount()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":     state,
		"project":   project,
		"documents": documentCount,
		"note":      fmt.Sprintf("Project %s is active with %d project documents indexed.", project.Name, documentCount),
	})
}
