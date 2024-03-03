package clients_pogos

import (
	"time"
)

type PromptResponse struct {
	Status       string
	ResponseRole string
	Response     string
	RequestId    uint64
}

type CohereGenerationResponse struct {
	ID          string `json:"id"`
	Generations []struct {
		ID           string `json:"id"`
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"generations"`
	Prompt string `json:"prompt"`
	Meta   struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
		BilledUnits struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"billed_units"`
	} `json:"meta"`
}

type CohereChatResponse struct {
	ResponseID   string `json:"response_id"`
	Text         string `json:"text"`
	GenerationID string `json:"generation_id"`
	TokenCount   struct {
		PromptTokens   int `json:"prompt_tokens"`
		ResponseTokens int `json:"response_tokens"`
		TotalTokens    int `json:"total_tokens"`
		BilledTokens   int `json:"billed_tokens"`
	} `json:"token_count"`
	Meta struct {
		APIVersion struct {
			Version string `json:"version"`
		} `json:"api_version"`
		BilledUnits struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"billed_units"`
	} `json:"meta"`
	ToolInputs any `json:"tool_inputs"`
}

type AnthropicPromptResponse struct {
	ID         string `json:"id"`
	Stop       string `json:"stop"`
	Type       string `json:"type"`
	Model      string `json:"model"`
	LogID      string `json:"log_id"`
	Completion string `json:"completion"`
	StopReason string `json:"stop_reason"`
}

type AnthropicChatResponse struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Type    string `json:"type"`
	Model   string `json:"model"`
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	StopReason   string `json:"stop_reason"`
	StopSequence any    `json:"stop_sequence"`
}

type GoogleChatResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		Index         int    `json:"index"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	PromptFeedback struct {
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"promptFeedback"`
}

type OpenAIResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Logprobs     struct {
			Content []struct {
				Bytes       []int   `json:"bytes"`
				Logprob     float64 `json:"logprob"`
				Token       string  `json:"token"`
				TopLogprobs []any   `json:"top_logprobs"`
			} `json:"content"`
		} `json:"logprobs"`
		Message struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created           int    `json:"created"`
	ID                string `json:"id"`
	Model             string `json:"model"`
	Object            string `json:"object"`
	SystemFingerprint any    `json:"system_fingerprint"`
	Usage             struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAIImageResponse struct {
	Data []struct {
		B64Json       *string `json:"b64_json"`
		RevisedPrompt *string `json:"revised_prompt"`
		Url           *string `json:"url"`
	} `json:"data"`
	Created int `json:"created"`
}

type ReplicateResponse struct {
	ID   string `json:"id"`
	Logs string `json:"logs"`
	Urls struct {
		Get    string `json:"get"`
		Cancel string `json:"cancel"`
	} `json:"urls"`
	Error any `json:"error"`
	Input struct {
		Prompt string `json:"prompt"`
	} `json:"input"`
	Output    []string  `json:"output"`
	Model     string    `json:"model"`
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type StabilityAIImageResponse struct {
	Artifacts []*struct {
		Base64       string `json:"base64"`
		Seed         uint64 `json:"seed"`
		FinishReason string `json:"finishReason"`
	} `json:"artifacts"`
}

type TogetherAIResponse[T any] struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created uint64   `json:"created"`
	Model   string   `json:"model"`
	Prompt  []string `json:"prompt"`
	Choices []*T     `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type TogetherAiImageChoice struct {
	ImageBase64 string      `json:"image_base64"`
	Logprobs    interface{} `json:"logprobs"`
}

type TogetherAITextChoice struct {
	Text         string      `json:"text"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     interface{} `json:"logprobs"`
}

type DeepInfraImageLegacyResponse struct {
	RequestId       string `json:"request_id"`
	InferenceStatus struct {
		Status          string      `json:"status"`
		Runtime         uint64      `json:"runtime_ms"`
		Cost            float64     `json:"cost"`
		TokensGenerated interface{} `json:"tokens_generated"`
		TokensInput     interface{} `json:"tokens_input"`
	} `json:"inference_status"`
	Images          []string `json:"images"`
	NotSafeDetected []bool   `json:"nsfw_content_detected"`
	Seed            uint64   `json:"seed"`
}

type DeepInfraImageResponse struct {
	RequestId       string `json:"request_id"`
	InferenceStatus struct {
		Status          string      `json:"status"`
		Runtime         uint64      `json:"runtime_ms"`
		Cost            float64     `json:"cost"`
		TokensGenerated interface{} `json:"tokens_generated"`
		TokensInput     interface{} `json:"tokens_input"`
	} `json:"inference_status"`
	Input struct {
		Prompt         string  `json:"prompt"`
		NegativePrompt string  `json:"negative_prompt"`
		Image          string  `json:"image"`
		Mask           string  `json:"mask"`
		Width          int     `json:"width"`
		Height         int     `json:"height"`
		Outputs        int     `json:"num_outputs"`
		Scheduler      string  `json:"scheduler"`
		InferenceSteps int     `json:"num_inference_steps"`
		GuidanceScale  float64 `json:"guidance_scale"`
		PromptStrength float64 `json:"prompt_strength"`
		Seed           uint64  `json:"seed"`
		Refine         string  `json:"refine"`
		HighNoiseFrac  float64 `json:"high_noise_frac"`
		RefineSteps    uint    `json:"refine_steps"`
		ApplyWaterMark bool    `json:"apply_watermark"`
	} `json:"input"`
	Output           []string `json:"output"`
	Id               string   `json:"id"`
	Logs             string   `json:"logs"`
	OutputFilePrefix string   `json:"output_file_prefix"`
	Status           string   `json:"status"`
	Metrics          struct {
		PredictTime float64 `json:"predict_time"`
	} `json:"metrics"`
}
