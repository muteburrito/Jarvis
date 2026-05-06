package server

import (
	"context"
	"net/http"
	"strings"

	"go-chatbot/internal/system"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":           "ok",
		"chat_model":       s.cfg.ChatModel,
		"embedding_model":  s.cfg.EmbeddingModel,
		"documents":        s.store.DocumentCount(),
		"chunks":           s.store.EntryCount(),
		"vector_dimension": s.store.Dimension(),
	}
	if s.cfg.VisionModel != "" {
		resp["vision_model"] = s.cfg.VisionModel
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAppConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app_name":        s.cfg.AppName,
		"support_email":   s.cfg.SupportEmail,
		"support_subject": s.cfg.SupportSubject,
		"support_url":     s.cfg.SupportURL,
	})
}
func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	status := system.GetSystemStatus(s.cfg.ChatModel, s.cfg.EmbeddingModel, s.cfg.VisionModel)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.chain.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list models")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"default": s.chain.DefaultChatModel(),
		"models":  s.chatModels(models),
	})
}

func (s *Server) chatModels(models []string) []string {
	filtered := make([]string, 0, len(models))
	for _, model := range models {
		if sameModel(model, s.cfg.EmbeddingModel) || sameModel(model, s.cfg.VisionModel) {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered
}

func (s *Server) isChatModel(ctx context.Context, model string) bool {
	models, err := s.chain.ListModels(ctx)
	if err != nil {
		return false
	}
	for _, chatModel := range s.chatModels(models) {
		if sameModel(chatModel, model) {
			return true
		}
	}
	return false
}

func sameModel(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return normalizeModelName(a) == normalizeModelName(b)
}

func normalizeModelName(model string) string {
	return strings.TrimSuffix(model, ":latest")
}
