package internal_anthropic_callers

import (
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type Anthropic struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

var (
	API_KEY            = "key"
	API_KEY_HEADER_KEY = "x-api-key"
	VERSION_HEADER_KEY = "anthropic-version"
	BETA_HEADER_KEY    = "anthropic-beta"
	API_URL            = "https://api.anthropic.com/v1"
	VERSION            = "2023-06-01"
	BETA               = "messages-2023-12-15"
	TIMEOUT            = 5 * time.Minute
)

func anthropicAI(logger commons.Logger, credential *integration_api.Credential) Anthropic {
	return Anthropic{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (aicaller *Anthropic) GetClient() (*anthropic.Client, error) {
	credentials := aicaller.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		aicaller.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}
	clt := anthropic.NewClient(
		option.WithAPIKey(cx.(string)),
	)
	return &clt, nil
}

func (anthropicC *Anthropic) UsageMetrics(usages anthropic.Usage) types.Metrics {
	metrics := make(types.Metrics, 0)
	metrics = append(metrics, &types.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.OutputTokens),
		Description: "Input token",
	})

	metrics = append(metrics, &types.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.InputTokens),
		Description: "Output Token",
	})

	metrics = append(metrics, &types.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.InputTokens+usages.OutputTokens),
		Description: "Total Token",
	})
	return metrics
}
