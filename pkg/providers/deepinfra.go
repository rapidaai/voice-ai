package providers

const DEEP_INFRA_STABILITY_AI_SDXL = "stability-ai/sdxl"

var v2ModelMap = map[string]bool{
	DEEP_INFRA_STABILITY_AI_SDXL: true,
}

func IsDeepInfraV2ImageModel(model string) bool {
	if _, ok := v2ModelMap[model]; ok {
		return true
	} else {
		return false
	}
}
