package internal_services

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
)

type agentLookupAssistantService struct {
	AssistantService
}

func (agentLookupAssistantService) GetAssistantWithPhoneDeploymentById(_ context.Context, agentId uint64) (*internal_assistant_entity.Assistant, error) {
	assistant := &internal_assistant_entity.Assistant{}
	assistant.Id = agentId
	return assistant, nil
}

func (agentLookupAssistantService) GetAssistantWithPhoneDeploymentByDID(_ context.Context, _ string) (*internal_assistant_entity.Assistant, error) {
	assistant := &internal_assistant_entity.Assistant{}
	assistant.Id = 42
	return assistant, nil
}

func TestAssistantServiceGetAssistantWithPhoneDeploymentById(t *testing.T) {
	var service AssistantService = agentLookupAssistantService{}

	assistant, err := service.GetAssistantWithPhoneDeploymentById(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetAssistantWithPhoneDeploymentById() error = %v", err)
	}
	if assistant.Id != 42 {
		t.Fatalf("GetAssistantWithPhoneDeploymentById() assistant ID = %d, want 42", assistant.Id)
	}
}

func TestAssistantServiceGetAssistantWithPhoneDeploymentByDID(t *testing.T) {
	var service AssistantService = agentLookupAssistantService{}

	assistant, err := service.GetAssistantWithPhoneDeploymentByDID(context.Background(), "+15551234567")
	if err != nil {
		t.Fatalf("GetAssistantWithPhoneDeploymentByDID() error = %v", err)
	}
	if assistant.Id != 42 {
		t.Fatalf("GetAssistantWithPhoneDeploymentByDID() assistant ID = %d, want 42", assistant.Id)
	}
}
