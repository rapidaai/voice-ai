package internal_openai_callers

import (
	"context"
	"fmt"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/caller"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type ModerationsCaller struct {
	OpenAI
}

func NewModerationsCaller(logger commons.Logger, credential *integration_api.Credential) internal_callers.ModerationsCaller {
	return &ModerationsCaller{
		OpenAI: openAI(logger, credential),
	}
}

func (stc *ModerationsCaller) GetModeration(ctx context.Context,
	content *types.Content, options *internal_callers.ModerationOptions) (*types.Content, types.Metrics, error) {
	//
	// Working with chat complition with vision
	//
	start := time.Now()
	// client, err := stc.GetClient()
	timeMetric := &types.Metric{
		Name:        type_enums.TIME_TAKEN.String(),
		Value:       fmt.Sprintf("%d", int64(time.Since(start))),
		Description: "Time taken to serve the llm request",
	}
	// if err != nil {
	// 	return nil, types.Metrics{timeMetric}, err
	// }
	// Will need moderation for chat in future

	return &types.Content{
		ContentType:   "text",
		ContentFormat: "raw",
	}, types.Metrics{timeMetric}, nil
}
