package internal_callers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	integration_api "github.com/rapidaai/protos"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
)

type Caller interface {
	Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error)
	CallWithPayload(ctx context.Context, endpoint, method string, headers map[string]string, payload io.Reader) (*string, error)
}

type AICaller struct {
	logger commons.Logger
}

// AIOptions holds configuration and hooks for handling an AI request.
// RequestId is a unique identifier for the request.
// PreHook is a function executed before the main AI logic, allowing manipulation of the result map `rst`.
// PostHook is a function executed after the AI processing, passing the result map `rst` and performance metrics `metrics`.
// ModelParameter holds a slice of model parameters used for AI processing, defined by the `integration_api.ModelParameter` type.
// This struct enables customizable pre- and post-processing of requests and results while carrying relevant AI configuration data.
type AIOptions struct {
	RequestId      uint64
	PreHook        func(rst map[string]interface{})
	PostHook       func(rst map[string]interface{}, metrics types.Metrics)
	ModelParameter map[string]*anypb.Any
}

type ModerationOptions struct {
	AIOptions
	Language    *string
	Filename    *string
	Temperature *float32
}

type CompletionOptions struct {
	AIOptions

	Version *string
}

type CredentialVerifierOptions struct {
	AIOptions
}

type Verifier interface {
	CredentialVerifier(
		ctx context.Context,
		options *CredentialVerifierOptions) (*string, error)
}

type LargeLanguageCaller interface {
	GetChatCompletion(
		ctx context.Context,
		allMessages []*lexatic_backend.Message,
		options *ChatCompletionOptions,
	) (*types.Message, types.Metrics, error)

	StreamChatCompletion(
		ctx context.Context,
		allMessages []*lexatic_backend.Message,
		options *ChatCompletionOptions,
		onStream func(types.Message) error,
		onMetrics func(*types.Message, types.Metrics) error,
		onError func(err error),
	) error
}

type EmbeddingCaller interface {
	GetEmbedding(ctx context.Context,
		content map[int32]string, options *EmbeddingOptions) ([]*integration_api.Embedding, types.Metrics, error)
}

type ModerationsCaller interface {
	GetModeration(ctx context.Context,
		content *types.Content, options *ModerationOptions) (*types.Content, types.Metrics, error)
}

type RerankingCaller interface {
	GetReranking(ctx context.Context,
		query string,
		content map[int32]*lexatic_backend.Content,
		options *RerankerOptions,
	) ([]*integration_api.Reranking, types.Metrics, error)
}

type CredentialResolver = func() map[string]interface{}

func (AICaller *AICaller) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	AICaller.logger.Debugf("making request to llm with %+v", req)
	return client.Do(req)
}

func Unmarshal(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}
