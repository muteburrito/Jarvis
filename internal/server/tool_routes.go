package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-chatbot/internal/workbench"
)

func (s *Server) handleSearchProjectFiles(w http.ResponseWriter, r *http.Request) {
	root, repoMap, ok := s.activeProjectWorkspace(w)
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results := workbench.SearchProjectFiles(repoMap, workbench.FileSearchOptions{
		Query: r.URL.Query().Get("query"),
		Kind:  r.URL.Query().Get("kind"),
		Limit: limit,
	})
	s.recordTaskTrace("tool_read", "Searched active project files", map[string]string{
		"root":    root,
		"query":   strings.TrimSpace(r.URL.Query().Get("query")),
		"results": fmt.Sprintf("%d", len(results)),
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"root":  root,
		"files": results,
	})
}

func (s *Server) handleReadProjectFile(w http.ResponseWriter, r *http.Request) {
	root, repoMap, ok := s.activeProjectWorkspace(w)
	if !ok {
		return
	}

	var req workbench.FileReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := workbench.ReadProjectFile(root, repoMap, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.recordTaskTrace("tool_read", "Read active project file", map[string]string{
		"path":      result.Path,
		"truncated": strconv.FormatBool(result.Truncated),
	})
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSummarizeProjectFile(w http.ResponseWriter, r *http.Request) {
	root, repoMap, ok := s.activeProjectWorkspace(w)
	if !ok {
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	summary, err := workbench.SummarizeProjectFile(root, repoMap, req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.recordTaskTrace("tool_read", "Summarized active project file", map[string]string{
		"path": summary.Path,
		"kind": summary.Kind,
	})
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleRunProjectCommand(w http.ResponseWriter, r *http.Request) {
	root, ok := s.activeProjectPath()
	if !ok {
		writeError(w, http.StatusNotFound, "open a project before running commands")
		return
	}

	var req workbench.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := workbench.RunApprovedCommand(root, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.recordTaskTrace("tool_command", "Ran approved project command", map[string]string{
		"command":   strings.TrimSpace(req.Command + " " + strings.Join(req.Args, " ")),
		"exit_code": strconv.Itoa(result.ExitCode),
		"timed_out": strconv.FormatBool(result.TimedOut),
	})
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) activeProjectWorkspace(w http.ResponseWriter) (string, *workbench.RepoMap, bool) {
	root, ok := s.activeProjectPath()
	if !ok {
		writeError(w, http.StatusNotFound, "open a project before using project tools")
		return "", nil, false
	}
	repoMap, err := workbench.LoadRepoMap(s.cfg.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "workspace map not available yet")
			return "", nil, false
		}
		writeError(w, http.StatusInternalServerError, "failed to load workspace map")
		return "", nil, false
	}
	if !sameCleanPath(root, repoMap.Root) {
		freshMap, err := workbench.ScanRepository(root)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan active project")
			return "", nil, false
		}
		_ = workbench.SaveRepoMap(s.cfg.DataDir, freshMap)
		repoMap = freshMap
	}
	return root, repoMap, true
}

func sameCleanPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
