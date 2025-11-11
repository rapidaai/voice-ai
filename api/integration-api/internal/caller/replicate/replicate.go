package internal_replicate_callers

import (
	"errors"
	"fmt"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
	replicate_go "github.com/replicate/replicate-go"
)

type Replicate struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

var (
	API_KEY            = "key"
	API_KEY_HEADER_KEY = "Authorization"
	API_URL            = "https://api.mistral.ai/"
	TIMEOUT            = 5 * time.Minute
)

func replicate(logger commons.Logger, credential *integration_api.Credential) Replicate {
	return Replicate{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}

func (replicate *Replicate) GetClient() (*replicate_go.Client, error) {
	credentials := replicate.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		replicate.logger.Errorf("Unable to get client for replicate")
		return nil, errors.New("unable to resolve the credential")
	}
	return replicate_go.NewClient(replicate_go.WithToken(cx.(string)))
}

func (replicate *Replicate) UsageMetrics(usages *replicate_go.PredictionMetrics) types.Metrics {

	metrics := make(types.Metrics, 0)
	if usages != nil {

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.PROVIDER_GENERATE_TIME.String(),
			Value:       fmt.Sprintf("%f", *usages.PredictTime),
			Description: "Time taken to generate by provider",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.PROVIDER_TOTAL_TIME.String(),
			Value:       fmt.Sprintf("%f", *usages.TotalTime),
			Description: "Total time taken by provider",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.TimeToFirstToken),
			Description: "Time to First Token",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TOKEN_PRE_SECOND.String(),
			Value:       fmt.Sprintf("%f", *usages.TokensPerSecond),
			Description: "Token Per second",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.InputTokenCount),
			Description: "Input token",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.OutputTokenCount),
			Description: "Output Token",
		})

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.InputTokenCount+*usages.OutputTokenCount),
			Description: "Total Token",
		})
	}
	return metrics
}
