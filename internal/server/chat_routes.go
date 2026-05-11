package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"

	"go-chatbot/internal/rag"
	"go-chatbot/internal/websearch"
)

type chatRequest struct {
	Query            string              `json:"query"`
	CodeSnippet      string              `json:"code_snippet"`
	CodeLang         string              `json:"code_language"`
	History          []ollamaapi.Message `json:"history"`
	Research         bool                `json:"research"`
	Locale           string              `json:"locale"`
	Timezone         string              `json:"timezone"`
	Model            string              `json:"model"`
	FocusDocumentIDs []string            `json:"focus_document_ids"`
	FocusFiles       []string            `json:"focus_files"`
	ReplyTo          *rag.ReplyContext   `json:"reply_to"`
}

type sseEvent struct {
	Token    string                   `json:"token"`
	Done     bool                     `json:"done"`
	Sources  []map[string]string      `json:"sources,omitempty"`
	Progress *websearch.ProgressEvent `json:"progress,omitempty"`
}

var urlPattern = regexp.MustCompile(`https?://[^\s<>"]+`)

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	query := strings.TrimSpace(req.Query)
	codeSnippet := strings.TrimSpace(req.CodeSnippet)
	codeLang := strings.TrimSpace(req.CodeLang)
	if query == "" && codeSnippet == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	effectiveQuery := buildChatQuery(query, codeSnippet, codeLang)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	var assistantResponse strings.Builder
	onToken := func(token string) error {
		assistantResponse.WriteString(token)
		data, _ := json.Marshal(sseEvent{Token: token})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}
	onProgress := func(progress websearch.ProgressEvent) error {
		data, _ := json.Marshal(sseEvent{Progress: &progress})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	}

	chatModel := strings.TrimSpace(req.Model)
	if chatModel == "" {
		chatModel = s.chain.DefaultChatModel()
	}
	if !s.isChatModel(r.Context(), chatModel) {
		data, _ := json.Marshal(sseEvent{Token: "\n\nError: selected model is not installed: " + chatModel, Done: true})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}
	s.recordTaskMessage("user", effectiveQuery)
	s.recordTaskTrace("chat", "Started chat request", map[string]string{
		"model":    chatModel,
		"research": fmt.Sprintf("%t", req.Research),
	})
	s.setTaskModel(chatModel)

	urls := extractURLs(query)
	if len(urls) > 0 {
		indexed := make(map[string]bool)
		for _, d := range s.store.ListDocuments() {
			indexed[d.FilePath] = true
		}
		for _, u := range urls {
			if indexed[u] {
				continue
			}
			_, title, chunks, err := s.processor.ProcessURL(r.Context(), u)
			if err != nil {
				slog.Warn("auto-fetch URL failed", "url", u, "error", err)
				s.recordTaskTrace("live_context", "URL fetch failed", map[string]string{"url": u})
				continue
			}
			slog.Info("auto-fetched URL from chat", "url", u, "title", title, "chunks", chunks)
			s.recordTaskTrace("live_context", "Fetched URL from chat", map[string]string{
				"url":    u,
				"title":  title,
				"chunks": fmt.Sprintf("%d", chunks),
			})
		}
	}

	if req.Research {
		s.recordTaskTrace("research", "Started research mode", nil)
		if err := s.researcher.Research(r.Context(), effectiveQuery, chatModel, onProgress); err != nil {
			slog.Warn("research failed, continuing with existing context", "error", err)
			s.recordTaskTrace("research", "Research failed", map[string]string{"error": err.Error()})
		}
	} else if shouldGatherLiveContext(query, codeSnippet) {
		liveCtx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		fetched, err := s.researcher.GatherLiveContext(liveCtx, effectiveQuery, chatModel)
		if err != nil {
			slog.Warn("quiet live context failed", "error", err)
			s.recordTaskTrace("live_context", "Quiet live context failed", map[string]string{"error": err.Error()})
		} else if fetched > 0 {
			s.recordTaskTrace("live_context", "Quietly refreshed live context", map[string]string{"pages": fmt.Sprintf("%d", fetched)})
		}
	}

	focusIDs, focusFiles := s.resolveFocusedDocuments(req.FocusDocumentIDs, req.FocusFiles)
	queryOpts := &rag.QueryOptions{
		ResearchMode:     req.Research,
		Locale:           req.Locale,
		Timezone:         req.Timezone,
		ChatModel:        chatModel,
		FocusDocumentIDs: focusIDs,
		FocusFiles:       focusFiles,
		ReplyTo:          req.ReplyTo,
	}
	result, err := s.chain.Query(r.Context(), effectiveQuery, req.History, onToken, queryOpts)
	if err != nil {
		slog.Error("chat query failed", "error", err)
		s.recordTaskTrace("error", "Chat query failed", map[string]string{"error": err.Error()})
		data, _ := json.Marshal(sseEvent{Token: "\n\nError: " + err.Error(), Done: true})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}
	s.recordTaskTrace("retrieval", "Prepared answer context", map[string]string{"sources": fmt.Sprintf("%d", len(result.Sources))})
	s.recordTaskMessage("assistant", assistantResponse.String())

	data, _ := json.Marshal(sseEvent{Done: true, Sources: result.Sources})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (s *Server) resolveFocusedDocuments(ids []string, names []string) ([]string, []string) {
	docs := s.store.ListDocuments()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			idSet[id] = true
		}
	}

	nameSet := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			nameSet[name] = true
		}
	}

	focusIDs := make([]string, 0)
	focusFiles := make([]string, 0)
	for _, doc := range docs {
		filename := strings.TrimSpace(doc.Filename)
		filePath := strings.TrimSpace(doc.FilePath)
		lowerFilename := strings.ToLower(filename)
		lowerPath := strings.ToLower(filePath)

		matched := idSet[doc.ID]
		if !matched && len(nameSet) > 0 {
			for name := range nameSet {
				if name == lowerFilename || name == lowerPath || strings.Contains(lowerFilename, name) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}

		focusIDs = append(focusIDs, doc.ID)
		if filename != "" {
			focusFiles = append(focusFiles, filename)
		} else if filePath != "" {
			focusFiles = append(focusFiles, filePath)
		}
	}
	return focusIDs, focusFiles
}

func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	var urls []string
	for _, m := range matches {
		m = strings.TrimRight(m, ".,)]>\"'")
		urls = append(urls, m)
	}
	return urls
}

func buildChatQuery(query, codeSnippet, codeLang string) string {
	if codeSnippet == "" {
		return query
	}
	if query == "" {
		query = "Please analyze this code snippet."
	}
	if codeLang == "" {
		codeLang = "text"
	}
	return query + "\n\nCode snippet:\n```" + codeLang + "\n" + codeSnippet + "\n```"
}

func shouldGatherLiveContext(query string, codeSnippet string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || strings.TrimSpace(codeSnippet) != "" || len(extractURLs(query)) > 0 {
		return false
	}
	currentTerms := []string{
		"current",
		"currently",
		"latest",
		"today",
		"tonight",
		"yesterday",
		"tomorrow",
		"this week",
		"this month",
		"news",
		"weather",
		"temperature",
		"forecast",
		"stock",
		"share price",
		"exchange rate",
		"price of",
		"who is the",
		"president of",
		"prime minister",
		"ceo of",
		"release date",
		"version",
		"changelog",
	}
	for _, term := range currentTerms {
		if strings.Contains(query, term) {
			return true
		}
	}
	return false
}
