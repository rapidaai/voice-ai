package internal_google_callers

import (
	"time"

	"google.golang.org/genai"
)

// Accumulator for Google Gemini streaming responses
type GoogleChatCompletionAccumulator struct {
	Candidates      []*genai.Candidate `json:"candidates,omitempty"`
	candidateStates []candidateStreamState
	justFinished    candidateStreamState

	ResponseID     string                                       `json:"responseId,omitempty"`
	CreateTime     time.Time                                    `json:"createTime,omitempty"`
	ModelVersion   string                                       `json:"modelVersion,omitempty"`
	PromptFeedback *genai.GenerateContentResponsePromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *genai.GenerateContentResponseUsageMetadata  `json:"usageMetadata,omitempty"`
}

// Stream state enum
type streamState int

const (
	streamStateEmpty streamState = iota
	streamStateContent
	streamStateFunctionCall
	streamStateFunctionResponse
	streamStateFinished
)

// Tracks state of individual candidate stream
type candidateStreamState struct {
	state streamState
	index int
}

// AddChunk incorporates a streamed GenerateContentResponse into the accumulator
func (acc *GoogleChatCompletionAccumulator) AddChunk(resp *genai.GenerateContentResponse) bool {
	acc.justFinished = candidateStreamState{}

	if acc.ResponseID == "" {
		acc.ResponseID = resp.ResponseID
		acc.CreateTime = resp.CreateTime
		acc.ModelVersion = resp.ModelVersion
		acc.PromptFeedback = resp.PromptFeedback
		acc.UsageMetadata = resp.UsageMetadata
	}

	for _, incoming := range resp.Candidates {
		index := int(incoming.Index)
		acc.Candidates = expandToFit(acc.Candidates, index)
		acc.candidateStates = expandToFit(acc.candidateStates, index)

		existing := acc.Candidates[index]
		if existing == nil {
			acc.Candidates[index] = &genai.Candidate{}
			existing = acc.Candidates[index]
		}

		// Accumulate Content Parts
		if incoming.Content != nil {
			if existing.Content == nil {
				existing.Content = &genai.Content{Role: incoming.Content.Role}
			}
			existing.Content.Parts = append(existing.Content.Parts, incoming.Content.Parts...)
		}

		// Copy other fields
		existing.FinishReason = incoming.FinishReason
		existing.FinishMessage = incoming.FinishMessage
		existing.TokenCount += incoming.TokenCount

		newState := detectState(incoming)
		prevState := acc.candidateStates[index]
		if prevState != newState {
			acc.justFinished = prevState
		}
		acc.candidateStates[index] = newState
	}

	return true
}

// detectState determines what type of stream state a Candidate represents
func detectState(c *genai.Candidate) candidateStreamState {
	if c.FinishReason != "" {
		return candidateStreamState{state: streamStateFinished, index: int(c.Index)}
	}
	if c.Content != nil && len(c.Content.Parts) > 0 {
		for _, p := range c.Content.Parts {
			if p.FunctionCall != nil {
				return candidateStreamState{state: streamStateFunctionCall, index: int(c.Index)}
			}
			if p.FunctionResponse != nil {
				return candidateStreamState{state: streamStateFunctionResponse, index: int(c.Index)}
			}
			if p.Text != "" {
				return candidateStreamState{state: streamStateContent, index: int(c.Index)}
			}
		}
	}
	return candidateStreamState{state: streamStateEmpty, index: int(c.Index)}
}

// JustFinishedText returns the most recently completed text content
func (acc *GoogleChatCompletionAccumulator) JustFinishedText() (string, bool) {
	if acc.justFinished.state != streamStateContent {
		return "", false
	}
	parts := acc.Candidates[acc.justFinished.index].Content.Parts
	var text string
	for _, p := range parts {
		text += p.Text
	}
	return text, true
}

// JustFinishedFunctionCall returns the most recently completed function call
func (acc *GoogleChatCompletionAccumulator) JustFinishedFunctionCall() (*genai.FunctionCall, bool) {
	if acc.justFinished.state != streamStateFunctionCall {
		return nil, false
	}
	for _, p := range acc.Candidates[acc.justFinished.index].Content.Parts {
		if p.FunctionCall != nil {
			return p.FunctionCall, true
		}
	}
	return nil, false
}

// JustFinishedFunctionResponse returns the most recently completed function response
func (acc *GoogleChatCompletionAccumulator) JustFinishedFunctionResponse() (*genai.FunctionResponse, bool) {
	if acc.justFinished.state != streamStateFunctionResponse {
		return nil, false
	}
	for _, p := range acc.Candidates[acc.justFinished.index].Content.Parts {
		if p.FunctionResponse != nil {
			return p.FunctionResponse, true
		}
	}
	return nil, false
}

// expandToFit resizes a slice to fit the given index
func expandToFit[T any](slice []T, index int) []T {
	if index < len(slice) {
		return slice
	}
	newSlice := make([]T, index+1)
	copy(newSlice, slice)
	return newSlice
}
