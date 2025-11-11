package integration_client_builders

import (
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type InputChatBuilder interface {
	Credential(i uint64, dp *structpb.Struct) *lexatic_backend.Credential

	Chat(
		credential *lexatic_backend.Credential,
		modelOpts map[string]*anypb.Any,
		tools []*lexatic_backend.FunctionDefinition,
		additionalData map[string]string,
		conversations ...*lexatic_backend.Message,
	) *lexatic_backend.ChatRequest

	Message(
		templates []*gorm_types.PromptTemplate,
		arguments map[string]interface{},
	) []*lexatic_backend.Message

	Options(
		opts map[string]interface{},
		options map[string]*anypb.Any,
	) map[string]*anypb.Any

	Arguments(
		variables []*gorm_types.PromptVariable,
		arguments map[string]*anypb.Any,
	) map[string]interface{}

	PromptArguments(
		variables []*gorm_types.PromptVariable,
	) map[string]interface{}
}

type InputEmbeddingBuilder interface {
	Credential(i uint64, dp *structpb.Struct) *lexatic_backend.Credential
	Embedding(
		credential *lexatic_backend.Credential,
		modelOpts map[string]*anypb.Any,
		additionalData map[string]string,
		contents map[int32]string,
	) *lexatic_backend.EmbeddingRequest
	Options(
		opts map[string]interface{},
		options map[string]*anypb.Any) map[string]*anypb.Any
}

type InputRerankingBuilder interface {
	Credential(i uint64, dp *structpb.Struct) *lexatic_backend.Credential
	Reranking(
		credential *lexatic_backend.Credential,
		modelOpts map[string]*anypb.Any,
		additionalData map[string]string,
		contents map[int32]*lexatic_backend.Content,
	) *lexatic_backend.RerankingRequest
	Options(
		opts map[string]interface{},
		options map[string]*anypb.Any) map[string]*anypb.Any
}
