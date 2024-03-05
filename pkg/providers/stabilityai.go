package providers

import (
	"strings"

	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

const STABILITY_AI_STABLE_DIFFUSION_XL_1024_V1_0 = "stable-diffusion-xl-1024-v1-0"

func formatStabilityAISizeParameter(params []*integration_api.ModelParameter, model string) []*integration_api.ModelParameter {
	if model != STABILITY_AI_STABLE_DIFFUSION_XL_1024_V1_0 {
		return params
	}
	// Hard coding for now
	var dimensions []string
	var indexToRemove *int
	for i, param := range params {
		if param.Key == "size" {
			indexToRemove = &i
			dimensions = strings.Split(param.Value, "x")
		}
	}

	if dimensions != nil && indexToRemove != nil {
		params = append(params, &integration_api.ModelParameter{
			Key:   "width",
			Value: dimensions[0],
			Type:  "integer",
		})
		params = append(params, &integration_api.ModelParameter{
			Key:   "height",
			Value: dimensions[1],
			Type:  "integer",
		})

		params = append(params[:*indexToRemove], params[*indexToRemove+1:]...)
	}
	return params
}

func FormatStabilityAIParameters(params []*integration_api.ModelParameter, model string) []*integration_api.ModelParameter {
	params = formatStabilityAISizeParameter(params, model)
	return params
}
