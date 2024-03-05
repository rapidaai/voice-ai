package clients_response_processors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lexatic/web-backend/config"
	clients "github.com/lexatic/web-backend/pkg/clients"
	integration_service_client "github.com/lexatic/web-backend/pkg/clients/integration"
	clients_pogos "github.com/lexatic/web-backend/pkg/clients/pogos"
	"github.com/lexatic/web-backend/pkg/commons"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

type chatResponseProcessor struct {
	cfg               *config.AppConfig
	logger            commons.Logger
	integrationClient clients.IntegrationServiceClient
}

func NewChatResponseProcessor(cfg *config.AppConfig, lgr commons.Logger) ResponseProcessor[[]*clients_pogos.Interaction] {
	return &chatResponseProcessor{logger: lgr, cfg: cfg, integrationClient: integration_service_client.NewIntegrationServiceClientGRPC(cfg, lgr)}
}

func (crp *chatResponseProcessor) Process(ctx context.Context, cr *clients_pogos.RequestData[[]*clients_pogos.Interaction]) *clients_pogos.PromptResponse {
	if res, err := crp.integrationClient.Converse(ctx, cr); err != nil {
		crp.logger.Errorf("error while processing the chat llm request %v", err)
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	} else {
		return crp.unmarshalChatResponse(res, cr.ProviderName)
	}

}

func (crp *chatResponseProcessor) unmarshalChatResponse(res *integration_api.ChatResponse, provider string) *clients_pogos.PromptResponse {
	switch providerName := strings.ToLower(provider); providerName {
	case "cohere":
		return crp.unmarshalCohereChat(res)
	case "anthropic":
		return crp.unmarshalAnthropicChat(res)
	case "replicate":
		return crp.unmarshalReplicateChat(res)
	case "google":
		return crp.unmarshalGoogleChat(res)
	case "togetherai":
		return crp.unmarshalTogetherAiChat(res)
	default:
		return crp.unmarshalOpenAiChat(res)
	}

}

/*
chat unmarshalling
*/

func (crp *chatResponseProcessor) unmarshalOpenAiChat(res *integration_api.ChatResponse) *clients_pogos.PromptResponse {
	if res.Success {
		openAiRes := clients_pogos.OpenAIResponse{}
		err := json.Unmarshal([]byte(*res.Response), &openAiRes)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status:       "SUCCESS",
			ResponseRole: openAiRes.Choices[len(openAiRes.Choices)-1].Message.Role,
			Response:     openAiRes.Choices[len(openAiRes.Choices)-1].Message.Content,
			RequestId:    res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  *res.ErrorMessage,
			RequestId: res.RequestId,
		}
	}
}

func (crp *chatResponseProcessor) unmarshalTogetherAiChat(res *integration_api.ChatResponse) *clients_pogos.PromptResponse {
	if res.Success {
		openAiRes := clients_pogos.OpenAIResponse{}
		err := json.Unmarshal([]byte(*res.Response), &openAiRes)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status:       "SUCCESS",
			ResponseRole: openAiRes.Choices[len(openAiRes.Choices)-1].Message.Role,
			Response:     openAiRes.Choices[len(openAiRes.Choices)-1].Message.Content,
			RequestId:    res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  *res.ErrorMessage,
			RequestId: res.RequestId,
		}
	}
}

func (crp *chatResponseProcessor) unmarshalAnthropicChat(resp *integration_api.ChatResponse) *clients_pogos.PromptResponse {

	if resp.Success {
		anthropicRes := clients_pogos.AnthropicChatResponse{}
		err := json.Unmarshal([]byte(*resp.Response), &anthropicRes)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			RequestId:    resp.RequestId,
			Status:       "SUCCESS",
			ResponseRole: anthropicRes.Role,
			Response:     anthropicRes.Content[len(anthropicRes.Content)-1].Text,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  *resp.ErrorMessage,
			RequestId: resp.RequestId,
		}
	}
}
func (crp *chatResponseProcessor) unmarshalCohereChat(resp *integration_api.ChatResponse) *clients_pogos.PromptResponse {
	if resp.Success {
		cohereResp := clients_pogos.CohereChatResponse{}
		err := json.Unmarshal([]byte(*resp.Response), &cohereResp)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			RequestId:    resp.RequestId,
			Status:       "SUCCESS",
			ResponseRole: "CHATBOT",
			Response:     cohereResp.Text,
		}

	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  *resp.ErrorMessage,
			RequestId: resp.RequestId,
		}
	}
}
func (crp *chatResponseProcessor) unmarshalReplicateChat(res *integration_api.ChatResponse) *clients_pogos.PromptResponse {
	if res.Success {
		rpt := clients_pogos.ReplicateResponse{}
		err := json.Unmarshal([]byte(*res.Response), &rpt)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status:       "SUCCESS",
			ResponseRole: "BOT",
			Response:     strings.Join(rpt.Output, ""),
			RequestId:    res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			ResponseRole: "",
			Response:     *res.ErrorMessage,
			RequestId:    res.RequestId,
		}
	}
}
func (crp *chatResponseProcessor) unmarshalGoogleChat(resp *integration_api.ChatResponse) *clients_pogos.PromptResponse {
	if resp.Success {
		googleResponse := clients_pogos.GoogleChatResponse{}
		err := json.Unmarshal([]byte(*resp.Response), &googleResponse)
		candidates := googleResponse.Candidates
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			RequestId:    resp.RequestId,
			Status:       "SUCCESS",
			ResponseRole: candidates[0].Content.Role,
			Response:     candidates[0].Content.Parts[len(candidates[0].Content.Parts)-1].Text,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  *resp.ErrorMessage,
			RequestId: resp.RequestId,
		}
	}
}
