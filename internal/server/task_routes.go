package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleTaskState(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusNotFound, "task state not available")
		return
	}
	writeJSON(w, http.StatusOK, s.tasks.Snapshot())
}

func (s *Server) handleAddTaskTrace(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusNotFound, "task state not available")
		return
	}
	var req struct {
		Type     string            `json:"type"`
		Summary  string            `json:"summary"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Summary) == "" {
		writeError(w, http.StatusBadRequest, "type and summary are required")
		return
	}
	s.tasks.AddTrace(workbench.TraceEvent{
		Type:     strings.TrimSpace(req.Type),
		Summary:  strings.TrimSpace(req.Summary),
		Metadata: req.Metadata,
	})
	writeJSON(w, http.StatusOK, s.tasks.Snapshot())
}

func (s *Server) handleAddTaskEdit(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusNotFound, "task state not available")
		return
	}
	var req struct {
		Path    string `json:"path"`
		Action  string `json:"action"`
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Path) == "" || strings.TrimSpace(req.Action) == "" {
		writeError(w, http.StatusBadRequest, "path and action are required")
		return
	}
	edit := workbench.EditRecord{
		Path:    strings.TrimSpace(req.Path),
		Action:  strings.TrimSpace(req.Action),
		Summary: strings.TrimSpace(req.Summary),
	}
	s.tasks.AddEdit(edit)
	if project, ok := s.activeProject(); ok {
		if err := workbench.RecordProjectEdit(s.cfg.DataDir, project, edit); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record project edit history")
			return
		}
	}
	s.recordTaskTrace("edit", "Recorded edit history", map[string]string{
		"path":   strings.TrimSpace(req.Path),
		"action": strings.TrimSpace(req.Action),
	})
	writeJSON(w, http.StatusOK, s.tasks.Snapshot())
}

func (s *Server) handleClearTaskState(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusNotFound, "task state not available")
		return
	}
	if err := s.tasks.Clear(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear task state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) recordTaskMessage(role, content string) {
	if s.tasks != nil {
		s.tasks.AddMessage(role, content)
	}
}

func (s *Server) recordTaskTrace(eventType, summary string, metadata map[string]string) {
	if s.tasks != nil {
		s.tasks.AddTrace(workbench.TraceEvent{
			Type:     eventType,
			Summary:  summary,
			Metadata: metadata,
		})
	}
}

func (s *Server) setTaskModel(model string) {
	if s.tasks != nil {
		s.tasks.SetModel(model)
	}
}
