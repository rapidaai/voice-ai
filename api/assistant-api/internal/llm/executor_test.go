package internal_llm

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	endpoint_client "github.com/rapidaai/pkg/clients/endpoint"
	integration_client "github.com/rapidaai/pkg/clients/integration"
	web_client "github.com/rapidaai/pkg/clients/web"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

type factoryTestCommunication struct {
	assistant *internal_assistant_entity.Assistant
}

func (communication *factoryTestCommunication) OnPacket(context.Context, ...internal_type.Packet) error {
	return nil
}

func (communication *factoryTestCommunication) IntegrationCaller() integration_client.IntegrationServiceClient {
	return nil
}

func (communication *factoryTestCommunication) VaultCaller() web_client.VaultClient {
	return nil
}

func (communication *factoryTestCommunication) DeploymentCaller() endpoint_client.DeploymentServiceClient {
	return nil
}

func (communication *factoryTestCommunication) Auth() *types.Authentication {
	return &types.Authentication{}
}

func (communication *factoryTestCommunication) GetSource() utils.RapidaSource {
	return utils.PhoneCall
}

func (communication *factoryTestCommunication) Assistant() (*internal_assistant_entity.Assistant, error) {
	return communication.assistant, nil
}

func (communication *factoryTestCommunication) GetMode() type_enums.MessageMode {
	return ""
}

func (communication *factoryTestCommunication) Conversation() (*internal_conversation_entity.AssistantConversation, error) {
	return &internal_conversation_entity.AssistantConversation{
		Audited:     gorm_model.Audited{Id: 100},
		AssistantId: 200,
		Identifier:  "caller",
	}, nil
}

func (communication *factoryTestCommunication) GetHistories() []internal_type.MessagePacket {
	return nil
}

func (communication *factoryTestCommunication) Metadata() map[string]interface{} {
	return nil
}

func (communication *factoryTestCommunication) GetArgs() map[string]interface{} {
	return nil
}

func (communication *factoryTestCommunication) GetOptions() utils.Option {
	return utils.Option{}
}

func (communication *factoryTestCommunication) GetKnowledge(context.Context, uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return nil, nil
}

func (communication *factoryTestCommunication) RetrieveToolKnowledge(
	context.Context,
	*internal_knowledge_gorm.Knowledge,
	string,
	string,
	map[string]interface{},
	*internal_type.KnowledgeRetrieveOption,
) ([]internal_type.KnowledgeContextResult, error) {
	return nil, nil
}

func (communication *factoryTestCommunication) IsConditionAllowed(utils.Option, string) bool {
	return true
}

func TestNewCreatesAgentflowExecutor(t *testing.T) {
	assistant := &internal_assistant_entity.Assistant{
		AssistantProvider: type_enums.AGENTFLOW,
		AssistantProviderAgentflow: &internal_assistant_entity.AssistantProviderAgentflow{
			SchemaVersion: "2026-07-06",
			Definition: gorm_types.PromptMap{
				"schemaVersion": "2026-07-06",
				"entryNodeId":   "chat-input-1",
				"nodes": []map[string]interface{}{
					{"id": "chat-input-1", "type": "chat-input", "label": "Chat Input"},
					{"id": "end-1", "type": "end", "label": "End Conversation"},
				},
				"edges": []map[string]interface{}{
					{"id": "edge-1", "source": "chat-input-1", "sourceHandle": "next", "target": "end-1"},
				},
			},
		},
	}

	executor, err := New(
		WithAssistant(assistant),
		WithCommunication(&factoryTestCommunication{assistant: assistant}),
	)

	require.NoError(t, err)
	require.Equal(t, "agentflow", executor.Name())
}

func TestNewRejectsAgentflowWithoutProviderConfiguration(t *testing.T) {
	executor, err := New(WithAssistant(&internal_assistant_entity.Assistant{
		AssistantProvider: type_enums.AGENTFLOW,
	}))

	require.Nil(t, executor)
	require.EqualError(t, err, "llm: agentflow provider configuration is required")
}
