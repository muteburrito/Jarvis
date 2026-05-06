package workbench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TaskState struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Status        string        `json:"status"`
	SelectedModel string        `json:"selected_model"`
	Messages      []TaskMessage `json:"messages"`
	Traces        []TraceEvent  `json:"traces"`
	EditHistory   []EditRecord  `json:"edit_history"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type TaskMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type TraceEvent struct {
	Type      string            `json:"type"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type EditRecord struct {
	Path      string    `json:"path"`
	Action    string    `json:"action"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskStore struct {
	mu   sync.Mutex
	dir  string
	task TaskState
}

func NewTaskStore(dir string) (*TaskStore, error) {
	store := &TaskStore{dir: dir}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		store.task = TaskState{
			ID:        "local",
			Title:     "Local task",
			Status:    "active",
			UpdatedAt: time.Now(),
		}
		if saveErr := store.saveLocked(); saveErr != nil {
			return nil, saveErr
		}
	}
	return store, nil
}

func (s *TaskStore) Snapshot() TaskState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.task
}

func (s *TaskStore) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task.Messages = append(s.task.Messages, TaskMessage{
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	})
	if len(s.task.Messages) > 100 {
		s.task.Messages = s.task.Messages[len(s.task.Messages)-100:]
	}
	s.task.UpdatedAt = time.Now()
	_ = s.saveLocked()
}

func (s *TaskStore) AddTrace(event TraceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	s.task.Traces = append(s.task.Traces, event)
	if len(s.task.Traces) > 250 {
		s.task.Traces = s.task.Traces[len(s.task.Traces)-250:]
	}
	s.task.UpdatedAt = time.Now()
	_ = s.saveLocked()
}

func (s *TaskStore) AddEdit(edit EditRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if edit.CreatedAt.IsZero() {
		edit.CreatedAt = time.Now()
	}
	s.task.EditHistory = append(s.task.EditHistory, edit)
	if len(s.task.EditHistory) > 250 {
		s.task.EditHistory = s.task.EditHistory[len(s.task.EditHistory)-250:]
	}
	s.task.UpdatedAt = time.Now()
	_ = s.saveLocked()
}

func (s *TaskStore) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task.SelectedModel = model
	s.task.UpdatedAt = time.Now()
	_ = s.saveLocked()
}

func (s *TaskStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task = TaskState{
		ID:        "local",
		Title:     "Local task",
		Status:    "active",
		UpdatedAt: time.Now(),
	}
	return s.saveLocked()
}

func (s *TaskStore) load() error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.task)
}

func (s *TaskStore) saveLocked() error {
	data, err := json.MarshalIndent(s.task, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), data, 0o644)
}

func (s *TaskStore) path() string {
	return filepath.Join(s.dir, "task_state.json")
}
