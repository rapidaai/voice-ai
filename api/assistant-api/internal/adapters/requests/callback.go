package internal_adapter_requests

import (
	"context"

	"github.com/rapidaai/pkg/types"
)

type LLMCallback interface {
	OnGeneration(ctx context.Context, messageid string, out *types.Message) error
	OnGenerationComplete(ctx context.Context, messageid string, out *types.Message, metrics []*types.Metric) error
}

/*
* Customization is an interface that defines methods for retrieving various
* customization components of a request. These components include arguments,
* metadata, options, and an agent prompt template.
*
* The interface provides a standardized way to access these customization
* elements, allowing for flexible and extensible request handling across
* different parts of the system.
*
* Methods:
* - GetArgs: Returns a map of arguments as protocol buffer Any messages.
* - GetMetadata: Returns a map of metadata as protocol buffer Any messages.
* - GetOptions: Returns a map of options as protocol buffer Any messages.
* - GetAgentTemplate: Returns an AgentPromptTemplate, which is used for
*   customizing agent prompts.
*
* Implementations of this interface can provide specific logic for how these
* customization elements are stored and retrieved, allowing for different
* request types to handle their custom data in unique ways while still
* conforming to a common interface.
 */
type Customization interface {
	GetArgs() map[string]interface{}
	GetMetadata() map[string]interface{}
	GetOptions() map[string]interface{}
}
