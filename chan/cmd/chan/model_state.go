package main

import (
	"sync"

	"github.com/channyeintun/chan/internal/api"
)

type activeModelState struct {
	mu      sync.RWMutex
	client  api.LLMClient
	modelID string
}

type activeSubagentModelState struct {
	mu      sync.RWMutex
	modelID string
}

func newActiveModelState(client api.LLMClient, modelID string) *activeModelState {
	return &activeModelState{client: client, modelID: modelID}
}

func (s *activeModelState) Get() (api.LLMClient, string) {
	if s == nil {
		return nil, ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client, s.modelID
}

func (s *activeModelState) Set(client api.LLMClient, modelID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
	s.modelID = modelID
}

func newActiveSubagentModelState(modelID string) *activeSubagentModelState {
	return &activeSubagentModelState{modelID: modelID}
}

func (s *activeSubagentModelState) Get() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modelID
}

func (s *activeSubagentModelState) Set(modelID string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modelID = modelID
}
