package endpoint_client_builders

import (
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
)

type inputInvokeBuilder struct {
	logger commons.Logger
}

func NewInputInvokeBuilder(logger commons.Logger) InputInvokeBuilder {
	return &inputInvokeBuilder{
		logger: logger,
	}
}

func (in *inputInvokeBuilder) Invoke(
	endpointDef *lexatic_backend.EndpointDefinition,
	args map[string]*anypb.Any,
	metadata map[string]*anypb.Any,
	options map[string]*anypb.Any,
) *lexatic_backend.InvokeRequest {
	request := &lexatic_backend.InvokeRequest{
		Endpoint: endpointDef,
		Args:     args,
		Metadata: metadata,
		Options:  options,
	}
	return request
}

func (in *inputInvokeBuilder) Arguments(
	opts map[string]interface{},
	options map[string]*anypb.Any) map[string]*anypb.Any {
	if options == nil {
		options = make(map[string]*anypb.Any)
	}
	for key, value := range opts {
		vl, err := utils.InterfaceToAnyValue(value)
		if err != nil {
			in.logger.Warnf("unable to encode the arguments in proto struct with error %+v", err)
			continue
		}
		options[key] = vl
	}

	return options
}

func (in *inputInvokeBuilder) Options(
	opts map[string]interface{},
	options map[string]*anypb.Any) map[string]*anypb.Any {

	// If options is nil, initialize it
	if options == nil {
		options = make(map[string]*anypb.Any)
	}

	for key, value := range opts {
		structValue, err := utils.InterfaceToAnyValue(value)
		if err != nil {
			continue
		}
		options[key] = structValue
	}

	return options
}

func (in *inputInvokeBuilder) Metadata(
	opts map[string]interface{},
	options map[string]*anypb.Any) map[string]*anypb.Any {

	// If options is nil, initialize it
	if options == nil {
		options = make(map[string]*anypb.Any)
	}

	for key, value := range opts {
		structValue, err := utils.InterfaceToAnyValue(value)
		if err != nil {
			continue
		}
		options[key] = structValue
	}

	return options
}
