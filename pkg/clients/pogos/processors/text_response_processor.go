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

type textResponseProcessor struct {
	cfg               *config.AppConfig
	logger            commons.Logger
	integrationClient clients.IntegrationServiceClient
}

func NewTextResponseProcessor(cfg *config.AppConfig, lgr commons.Logger) ResponseProcessor[string] {
	return &textResponseProcessor{logger: lgr, cfg: cfg, integrationClient: integration_service_client.NewIntegrationServiceClientGRPC(cfg, lgr)}
}

func (trp *textResponseProcessor) Process(ctx context.Context, cr *clients_pogos.RequestData[string]) *clients_pogos.PromptResponse {
	if res, err := trp.integrationClient.Prompt(ctx, cr); err != nil {
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			Response:     err.Error(),
			ResponseRole: "assitant",
		}
	} else {
		return trp.unmarshalTextResponse(res, cr.ProviderName)
	}

}

func (trp *textResponseProcessor) unmarshalTextResponse(res *integration_api.GenerateResponse, provider string) *clients_pogos.PromptResponse {
	switch providerName := strings.ToLower(provider); providerName {
	case "cohere":
		return trp.unmarshalCohereText(res)
	case "anthropic":
		return trp.unmarshalAnthropicText(res)
	case "replicate":
		return trp.unmarshalReplicateText(res)
	case "google":
		return trp.unmarshalGoogleText(res)
	default:
		return trp.unmarshalOpenAiText(res)
	}
}
func (trp *textResponseProcessor) unmarshalCohereText(res *integration_api.GenerateResponse) *clients_pogos.PromptResponse {
	if res.Success {
		cohereResp := clients_pogos.CohereGenerationResponse{}
		err := json.Unmarshal([]byte(*res.Response), &cohereResp)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status:    "SUCCESS",
			Response:  cohereResp.Generations[0].Text,
			RequestId: res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  res.ErrorMessage,
			RequestId: res.RequestId,
		}
	}
}
func (trp *textResponseProcessor) unmarshalOpenAiText(res *integration_api.GenerateResponse) *clients_pogos.PromptResponse {
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
			Status:       "FAILURE",
			ResponseRole: "",
			Response:     res.ErrorMessage,
			RequestId:    res.RequestId,
		}
	}
}
func (trp *textResponseProcessor) unmarshalAnthropicText(res *integration_api.GenerateResponse) *clients_pogos.PromptResponse {
	if res.Success {
		ath := clients_pogos.AnthropicPromptResponse{}
		err := json.Unmarshal([]byte(*res.Response), &ath)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status: "SUCCESS",
			// ResponseRole:    openAiRes.Completion,
			Response:  ath.Completion,
			RequestId: res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:    "FAILURE",
			Response:  res.ErrorMessage,
			RequestId: res.RequestId,
		}
	}
}
func (trp *textResponseProcessor) unmarshalReplicateText(res *integration_api.GenerateResponse) *clients_pogos.PromptResponse {
	if res.Success {
		rpt := clients_pogos.ReplicateResponse{}
		err := json.Unmarshal([]byte(*res.Response), &rpt)
		if err != nil {
			fmt.Printf("%v", err)
		}
		return &clients_pogos.PromptResponse{
			Status:       "SUCCESS",
			ResponseRole: "",
			Response:     strings.Join(rpt.Output, ""),
			RequestId:    res.RequestId,
		}
	} else {
		return &clients_pogos.PromptResponse{
			Status:       "FAILURE",
			ResponseRole: "",
			Response:     res.ErrorMessage,
			RequestId:    res.RequestId,
		}
	}
}
func (trp *textResponseProcessor) unmarshalGoogleText(resp *integration_api.GenerateResponse) *clients_pogos.PromptResponse {
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
			Response:  resp.ErrorMessage,
			RequestId: resp.RequestId,
		}
	}
}
