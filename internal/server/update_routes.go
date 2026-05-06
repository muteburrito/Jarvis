package server

import (
	"context"
	"log/slog"
	"net/http"
)

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.updater.Status())
}

func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	st := s.updater.Status()
	if !st.UpdateAvailable {
		writeError(w, http.StatusConflict, "no update available")
		return
	}
	go func() {
		if err := s.updater.Apply(context.Background()); err != nil {
			slog.Error("apply update failed", "error", err)
		}
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "apply started"})
}
