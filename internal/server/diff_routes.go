package server

import (
	"net/http"
	"os"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleDiffSummary(w http.ResponseWriter, r *http.Request) {
	root, ok := s.activeProjectPath()
	if !ok {
		writeError(w, http.StatusNotFound, "open a git project to view local changes")
		return
	}

	summary, err := workbench.LoadDiffSummary(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) activeProjectPath() (string, bool) {
	state, err := workbench.LoadProjectState(s.cfg.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false
		}
		return "", false
	}
	for _, project := range state.Projects {
		if project.Active && project.Path != "" {
			return project.Path, true
		}
	}
	return "", false
}
