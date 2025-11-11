package endpoint_client_builders

import (
	lexatic_backend "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/anypb"
)

type InputInvokeBuilder interface {
	Invoke(
		endpointDef *lexatic_backend.EndpointDefinition,
		args map[string]*anypb.Any,
		metadata map[string]*anypb.Any,
		options map[string]*anypb.Any,
	) *lexatic_backend.InvokeRequest
	Options(
		opts map[string]interface{},
		options map[string]*anypb.Any) map[string]*anypb.Any
	Metadata(
		opts map[string]interface{},
		options map[string]*anypb.Any) map[string]*anypb.Any
	Arguments(
		opts map[string]interface{},
		arguments map[string]*anypb.Any) map[string]*anypb.Any
}
