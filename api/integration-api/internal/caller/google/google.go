package internal_google_callers

import (
	"context"
	"errors"
	"fmt"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/genai"
)

type Google struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

var (
	API_KEY = "key"
)

func google(logger commons.Logger, credential *integration_api.Credential) Google {
	return Google{
		logger: logger,
		credential: func() map[string]interface{} {
			return credential.GetValue().AsMap()
		},
	}
}
func (goog *Google) GetClient() (*genai.Client, error) {
	// need to replace with current request context
	ctx := context.Background()
	credentials := goog.credential()
	cx, ok := credentials[API_KEY]
	if !ok {
		goog.logger.Errorf("Unable to get client for user")
		return nil, errors.New("unable to resolve the credential")
	}
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cx.(string),
	})

}
func (goog *Google) UsageMetrics(usages *genai.GenerateContentResponseUsageMetadata) types.Metrics {
	metrics := make(types.Metrics, 0)
	if usages != nil {
		metrics = append(metrics, &types.Metric{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.PromptTokenCount),
			Description: "Input tokens (including cached content)",
		})

		if usages.CachedContentTokenCount > 0 {
			metrics = append(metrics, &types.Metric{
				Name:        "CACHED_CONTENT_TOKEN",
				Value:       fmt.Sprintf("%d", usages.CachedContentTokenCount),
				Description: "Cached content tokens",
			})
		}

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.CandidatesTokenCount),
			Description: "Output tokens",
		})

		if usages.ToolUsePromptTokenCount > 0 {
			metrics = append(metrics, &types.Metric{
				Name:        "TOOL_USE_PROMPT_TOKEN",
				Value:       fmt.Sprintf("%d", usages.ToolUsePromptTokenCount),
				Description: "Tool-use prompt tokens",
			})
		}

		if usages.ThoughtsTokenCount > 0 {
			metrics = append(metrics, &types.Metric{
				Name:        "THOUGHTS_TOKEN",
				Value:       fmt.Sprintf("%d", usages.ThoughtsTokenCount),
				Description: "Thoughts tokens for thinking models",
			})
		}

		metrics = append(metrics, &types.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokenCount),
			Description: "Total tokens (prompt, response, and tool-use)",
		})
	}
	return metrics
}
