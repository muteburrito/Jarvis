package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	if s.chats == nil {
		writeError(w, http.StatusNotFound, "chat store not available")
		return
	}
	writeJSON(w, http.StatusOK, s.chats.List())
}

func (s *Server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	if s.chats == nil {
		writeError(w, http.StatusNotFound, "chat store not available")
		return
	}
	var req struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	session, err := s.chats.Create(req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create chat")
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	if s.chats == nil {
		writeError(w, http.StatusNotFound, "chat store not available")
		return
	}
	session, ok := s.chats.Get(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleUpdateChat(w http.ResponseWriter, r *http.Request) {
	if s.chats == nil {
		writeError(w, http.StatusNotFound, "chat store not available")
		return
	}
	var req struct {
		Title    string                  `json:"title"`
		Messages []workbench.ChatMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	session, ok, err := s.chats.Update(chi.URLParam(r, "id"), req.Title, req.Messages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update chat")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	if s.chats == nil {
		writeError(w, http.StatusNotFound, "chat store not available")
		return
	}
	ok, err := s.chats.Delete(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete chat")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
