package internal_openai_callers

import (
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type OpenAI struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

var (
	DEFUALT_URL         = "https://api.openai.com/v1"
	API_URL             = "url"
	API_KEY             = "key"
	AZ_ENDPOINT_KEY     = "endpoint"
	AZ_SUBSCRIPTION_KEY = "subscription_key"
)

const (
	// ChatRoleAssistant - The role that provides responses to system-instructed, user-prompted input.
	ChatRoleAssistant string = "assistant"
	// ChatRoleFunction - The role that provides function results for chat completions.
	ChatRoleFunction string = "function"
	// ChatRoleSystem - The role that instructs or sets the behavior of the assistant.
	ChatRoleSystem string = "system"
	// ChatRoleTool - The role that represents extension tool activity within a chat completions operation.
	ChatRoleTool string = "tool"
	// ChatRoleUser - The role that provides input for chat completions.
	ChatRoleUser string = "user"
)

func openAI(logger commons.Logger, credential *integration_api.Credential) OpenAI {
	_credential := credential.GetValue().AsMap()
	return OpenAI{logger: logger,
		credential: func() map[string]interface{} {
			return _credential
		}}
}

func (openAI *OpenAI) GetClient() (*openai.Client, error) {
	openAI.logger.Debugf("Getting client for open ai")
	credentials := openAI.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		openAI.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}
	clt := openai.NewClient(
		option.WithAPIKey(cx.(string)),
	)
	return &clt, nil
}

func (openAI *OpenAI) GetComplitionUsages(usages openai.CompletionUsage) types.Metrics {
	metrics := make(types.Metrics, 0)
	metrics = append(metrics, &types.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.CompletionTokens),
		Description: "Input token",
	})

	metrics = append(metrics, &types.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.PromptTokens),
		Description: "Output Token",
	})

	metrics = append(metrics, &types.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.TotalTokens),
		Description: "Total Token",
	})
	return metrics
}
