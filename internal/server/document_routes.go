package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes)

	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "file too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	filename := sanitizeFilename(header.Filename)
	destPath := filepath.Join(s.cfg.DataDir, filename)

	out, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}
	out.Close()

	if !s.processor.CanLoad(destPath) {
		os.Remove(destPath)
		writeError(w, http.StatusBadRequest, "binary file not supported")
		return
	}

	docID, chunks, err := s.processor.ProcessFile(r.Context(), destPath)
	if err != nil {
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "failed to process file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       docID,
		"filename": filename,
		"chunks":   chunks,
		"status":   "processed",
	})
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	docs := s.store.ListDocuments()
	writeJSON(w, http.StatusOK, docs)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "document id required")
		return
	}

	var found bool
	for _, d := range s.store.ListDocuments() {
		if d.ID == id {
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}

	if err := s.processor.RemoveDocument(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove document")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (s *Server) handleIngestPath(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
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

	processed, chunks, err := s.processor.ProcessDirectory(r.Context(), req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process directory: "+err.Error())
		return
	}
	repoMap, err := workbench.ScanRepository(req.Path)
	if err != nil {
		slog.Warn("failed to build repo map", "path", req.Path, "error", err)
	} else if err := workbench.SaveRepoMap(s.cfg.DataDir, repoMap); err != nil {
		slog.Warn("failed to save repo map", "path", req.Path, "error", err)
	} else {
		s.recordTaskTrace("repo_map", "Updated repo map", map[string]string{
			"root":    repoMap.Root,
			"files":   fmt.Sprintf("%d", repoMap.FileCount),
			"symbols": fmt.Sprintf("%d", repoMap.SymbolCount),
		})
	}
	if err := workbench.SaveWatchedFolder(s.cfg.DataDir, req.Path); err != nil {
		slog.Warn("failed to save watched folder", "path", req.Path, "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"processed": processed,
		"chunks":    chunks,
		"path":      req.Path,
	})
}

func (s *Server) handleRepoMap(w http.ResponseWriter, r *http.Request) {
	repoMap, err := workbench.LoadRepoMap(s.cfg.DataDir)
	if err != nil {
		writeError(w, http.StatusNotFound, "repo map not found")
		return
	}
	writeJSON(w, http.StatusOK, repoMap)
}
func (s *Server) handleClearAll(w http.ResponseWriter, r *http.Request) {
	if err := s.processor.ClearAll(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear vectorstore")
		return
	}
	slog.Info("vectorstore cleared by user")
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}
func (s *Server) handleFetchURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		writeError(w, http.StatusBadRequest, "url must start with http:// or https://")
		return
	}

	docID, title, chunks, err := s.processor.ProcessURL(r.Context(), req.URL)
	if err != nil {
		slog.Error("failed to fetch URL", "url", req.URL, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch URL: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":       docID,
		"filename": title,
		"chunks":   chunks,
		"url":      req.URL,
		"status":   "processed",
	})
}
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		"..", "",
		"/", "",
		"\\", "",
		"\x00", "",
	)
	return replacer.Replace(name)
}
