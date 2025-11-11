package internal_cohere_callers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	cohereV2 "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

var (
	API_KEY = "key"
)

type Cohere struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

func NewCohere(logger commons.Logger, credential *integration_api.Credential) Cohere {
	return Cohere{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (cohere *Cohere) GetClient() (*cohereclient.Client, error) {
	credentials := cohere.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		cohere.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}
	return cohereclient.NewClient(
		cohereclient.WithToken(cx.(string)),
		cohereclient.WithHTTPClient(
			&http.Client{
				Timeout: time.Minute,
			},
		),
	), nil
}
func (cohere *Cohere) UsageMetrics(usages *cohereV2.Usage) types.Metrics {
	metrics := make(types.Metrics, 0)
	if usages != nil {
		if usages.Tokens.InputTokens != nil {
			metrics = append(metrics, &types.Metric{
				Name:        type_enums.OUTPUT_TOKEN.String(),
				Value:       fmt.Sprintf("%f", *usages.Tokens.InputTokens),
				Description: "Input token",
			})
		}

		if usages.Tokens.OutputTokens != nil {
			metrics = append(metrics, &types.Metric{
				Name:        type_enums.INPUT_TOKEN.String(),
				Value:       fmt.Sprintf("%f", *usages.Tokens.OutputTokens),
				Description: "Output Token",
			})
		}
		if usages.Tokens.OutputTokens != nil && usages.Tokens.InputTokens != nil {
			metrics = append(metrics, &types.Metric{
				Name:        type_enums.TOTAL_TOKEN.String(),
				Value:       fmt.Sprintf("%f", *usages.Tokens.InputTokens+*usages.Tokens.OutputTokens),
				Description: "Total Token",
			})
		}
	}
	return metrics
}
