package server

import (
	"fmt"

	"go-chatbot/internal/document"
	"go-chatbot/internal/vectorstore"
	"go-chatbot/internal/workbench"
)

func (s *Server) projectProcessor(project workbench.Project) (*document.Processor, *vectorstore.Store, string, error) {
	store, dir, err := s.projectStore(project)
	if err != nil {
		return nil, nil, "", err
	}
	return s.processor.WithStore(store, dir), store, dir, nil
}

func (s *Server) projectStore(project workbench.Project) (*vectorstore.Store, string, error) {
	dir := workbench.ResolveProjectVectorStoreDir(s.cfg.DataDir, project)
	store, err := vectorstore.LoadFromDisk(dir)
	if err != nil {
		return nil, "", fmt.Errorf("load project vector store: %w", err)
	}
	if store == nil {
		store = vectorstore.New(s.store.Dimension())
	}
	return store, dir, nil
}

func (s *Server) activeProjectStore() (*vectorstore.Store, string, bool) {
	state, err := workbench.LoadProjectState(s.cfg.DataDir)
	if err != nil {
		return nil, "", false
	}
	project, ok := workbench.ActiveProject(state)
	if !ok {
		return nil, "", false
	}
	store, dir, err := s.projectStore(project)
	if err != nil || store == nil {
		return nil, "", false
	}
	return store, dir, true
}

func (s *Server) activeProjectID() string {
	state, err := workbench.LoadProjectState(s.cfg.DataDir)
	if err != nil {
		return ""
	}
	project, ok := workbench.ActiveProject(state)
	if !ok {
		return ""
	}
	return project.ID
}
