package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ChatSession struct {
	ID        string        `json:"id"`
	ProjectID string        `json:"project_id,omitempty"`
	Title     string        `json:"title"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type ChatMessage struct {
	Role             string              `json:"role"`
	Content          string              `json:"content"`
	Sources          []map[string]string `json:"sources,omitempty"`
	Progress         []ProgressSnapshot  `json:"progress,omitempty"`
	Attachments      []AttachmentInfo    `json:"attachments,omitempty"`
	ReplyTo          *ReplyInfo          `json:"replyTo,omitempty"`
	FocusedDocuments []FocusedDocument   `json:"focusedDocuments,omitempty"`
	StartedAt        string              `json:"startedAt,omitempty"`
	CompletedAt      string              `json:"completedAt,omitempty"`
	DurationMs       int64               `json:"durationMs,omitempty"`
	Rating           string              `json:"rating,omitempty"`
}

type ProgressSnapshot struct {
	Step   string `json:"step"`
	Detail string `json:"detail"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

type AttachmentInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type ReplyInfo struct {
	Index   int    `json:"index,omitempty"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Excerpt string `json:"excerpt,omitempty"`
}

type FocusedDocument struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	FilePath string `json:"file_path,omitempty"`
}

type ChatSummary struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id,omitempty"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ChatStore struct {
	mu       sync.Mutex
	path     string
	sessions map[string]ChatSession
}

func NewChatStore(dir string) (*ChatStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := &ChatStore{
		path:     filepath.Join(dir, "chat_sessions.json"),
		sessions: make(map[string]ChatSession),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

func (s *ChatStore) List(projectID string) []ChatSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	summaries := make([]ChatSummary, 0, len(s.sessions))
	filterProjectID := cleanProjectID(projectID)
	for _, session := range s.sessions {
		sessionProjectID := cleanProjectID(session.ProjectID)
		if filterProjectID == "" && sessionProjectID != "" {
			continue
		}
		if filterProjectID != "" && sessionProjectID != "" && sessionProjectID != filterProjectID {
			continue
		}
		summaries = append(summaries, ChatSummary{
			ID:           session.ID,
			ProjectID:    session.ProjectID,
			Title:        session.Title,
			MessageCount: len(session.Messages),
			UpdatedAt:    session.UpdatedAt,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries
}

func (s *ChatStore) Create(title string, projectID string) (ChatSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	session := ChatSession{
		ID:        "chat_" + uuid.New().String()[:8],
		ProjectID: cleanProjectID(projectID),
		Title:     cleanChatTitle(title),
		Messages:  []ChatMessage{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.sessions[session.ID] = session
	return session, s.saveLocked()
}

func (s *ChatStore) Get(id string) (ChatSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	return session, ok
}

func (s *ChatStore) Update(id, title string, messages []ChatMessage) (ChatSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return ChatSession{}, false, nil
	}
	session.Messages = messages
	if strings.TrimSpace(title) != "" {
		session.Title = cleanChatTitle(title)
	} else {
		session.Title = deriveChatTitle(messages)
	}
	session.UpdatedAt = time.Now()
	s.sessions[id] = session
	return session, true, s.saveLocked()
}

func (s *ChatStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		return false, nil
	}
	delete(s.sessions, id)
	return true, s.saveLocked()
}

func (s *ChatStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var sessions []ChatSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return err
	}
	for _, session := range sessions {
		s.sessions[session.ID] = session
	}
	return nil
}

func (s *ChatStore) saveLocked() error {
	sessions := make([]ChatSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func deriveChatTitle(messages []ChatMessage) string {
	for _, message := range messages {
		if message.Role == "user" && strings.TrimSpace(message.Content) != "" {
			return cleanChatTitle(message.Content)
		}
	}
	return "New chat"
}

func cleanChatTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return "New chat"
	}
	if len(title) > 60 {
		return title[:57] + "..."
	}
	return title
}

func cleanProjectID(projectID string) string {
	return strings.TrimSpace(projectID)
}
