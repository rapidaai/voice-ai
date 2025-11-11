package integration_client_builders

import (
	"github.com/rapidaai/pkg/commons"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/parsers"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type inputChatBuilder struct {
	logger         commons.Logger
	templateParser parsers.StringTemplateParser
}

func NewChatInputBuilder(logger commons.Logger) InputChatBuilder {
	return &inputChatBuilder{
		logger:         logger,
		templateParser: parsers.NewPongo2StringTemplateParser(logger),
	}
}

func (in *inputChatBuilder) Credential(i uint64, dp *structpb.Struct) *lexatic_backend.Credential {
	return &lexatic_backend.Credential{
		Id:    i,
		Value: dp,
	}
}

func (in *inputChatBuilder) Chat(
	credential *lexatic_backend.Credential,
	modelOpts map[string]*anypb.Any,
	tools []*lexatic_backend.FunctionDefinition,
	additionalData map[string]string,
	conversations ...*lexatic_backend.Message,
) *lexatic_backend.ChatRequest {

	request := &lexatic_backend.ChatRequest{
		Credential:      credential,
		Conversations:   conversations,
		ModelParameters: modelOpts,
		AdditionalData:  additionalData,
	}
	for _, tl := range tools {
		if request.ToolDefinitions == nil {
			request.ToolDefinitions = make([]*lexatic_backend.ToolDefinition, 0)
		}
		request.ToolDefinitions = append(request.ToolDefinitions, &lexatic_backend.ToolDefinition{
			Type:               "function",
			FunctionDefinition: tl,
		})
	}
	return request
}

func (in *inputChatBuilder) WithinMessage(role, prompt string) *lexatic_backend.Message {
	return &lexatic_backend.Message{
		Role: role,
		Contents: []*lexatic_backend.Content{{
			ContentType:   commons.TEXT_CONTENT.String(),
			ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
			Content:       []byte(prompt),
		}},
	}
}

func (in *inputChatBuilder) Message(
	template []*gorm_types.PromptTemplate,
	arguments map[string]interface{}) []*lexatic_backend.Message {
	msg := make([]*lexatic_backend.Message, 0)
	for _, v := range template {
		content := in.templateParser.Parse(v.GetContent(), arguments)
		msg = append(msg, in.WithinMessage(v.GetRole(), content))
	}
	return msg
}

func (in *inputChatBuilder) Arguments(
	variables []*gorm_types.PromptVariable,
	arguments map[string]*anypb.Any) map[string]interface{} {
	args, err := utils.AnyMapToInterfaceMap(arguments)
	if err != nil {
	}
	existing := in.PromptArguments(variables)
	return utils.MergeMaps(existing, args)
}

func (in *inputChatBuilder) PromptArguments(
	variables []*gorm_types.PromptVariable,
) map[string]interface{} {
	existing := make(map[string]interface{}, 0)
	for _, v := range variables {
		existing[v.Name] = v.DefaultValue
	}
	return existing
}

func (in *inputChatBuilder) Options(
	opts map[string]interface{},
	options map[string]*anypb.Any) map[string]*anypb.Any {

	// If options is nil, initialize it
	if options == nil {
		options = make(map[string]*anypb.Any)
	}

	for key, value := range opts {
		structValue, err := structpb.NewValue(value)
		if err != nil {
			continue
		}
		anyValue, err := anypb.New(structValue)
		if err != nil {
			continue
		}

		options[key] = anyValue
	}

	return options
}
