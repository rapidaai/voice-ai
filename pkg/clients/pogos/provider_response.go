package clients_pogos

import (
	"encoding/json"
	"time"
)

// need to work on

type ProviderError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	RequestId  uint64 `json:"requestId"`
	Type       string `json:"type"`
}

func (e ProviderError) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		return "undefined error"
	}
	return string(b)
}

type PromptResponse struct {
	Status       string
	ResponseRole string
	Response     string
	RequestId    uint64
}

type CohereGenerationResponse struct {
	Id          string             `json:"id"`
	Prompt      string             `json:"prompt"`
	Generations []CohereGeneration `json:"generations"`
	CohereMeta  CohereMeta         `json:"meta"`
}

type CohereGeneration struct {
	Id           string `json:"id"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type CohereChatResponse struct {
	ResponseId   string `json:"response_id"`
	Text         string `json:"text"`
	GenerationId string `json:"generation_id"`
}

type CohereTokenCount struct {
	PromptTokens   string `json:"prompt_tokens"`
	ResponseTokens string `json:"response_tokens"`
	TotalTokens    string `json:"total_tokens"`
	BilledTokens   string `json:"billed_tokens"`
	CohereMeta     string `json:"meta"`
}

type CohereMeta struct {
	ApiVersion  CohereApiVersion  `json:"api_version"`
	BilledUnits CohereBilledUnits `json:"billed_units"`
}

type CohereApiVersion struct {
	Version string `json:"version"`
}

type CohereBilledUnits struct {
	InputTokens  uint64 `json:"input_tokens"`
	OutputTokens uint64 `json:"output_tokens"`
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
	ID    string `json:"id"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens      int `json:"total_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Object  string `json:"object"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Logprobs     any    `json:"logprobs"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Created           int `json:"created"`
	SystemFingerprint any `json:"system_fingerprint"`
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
