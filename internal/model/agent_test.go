package model

import (
	"testing"
	"time"
)

func TestAgent_EntityInterface(t *testing.T) {
	agent := &Agent{
		ID:        "test-agent",
		Namespace: "test-namespace",
		Metadata: map[string]interface{}{
			"key": "value",
		},
		SecretHash: []byte("hash"),
		LastSeen:   time.Now(),
		Status:     AgentStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Тест GetID
	if agent.GetID() != "test-agent" {
		t.Errorf("GetID() = %v, хотим 'test-agent'", agent.GetID())
	}

	// Тест SetID
	agent.SetID("new-agent-id")
	if agent.GetID() != "new-agent-id" {
		t.Errorf("После SetID() GetID() = %v, хотим 'new-agent-id'", agent.GetID())
	}
}
