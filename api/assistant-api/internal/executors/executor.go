package internal_executors

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	"github.com/rapidaai/pkg/types"
	lexatic_backend "github.com/rapidaai/protos"
)

/*
AssistantExecutor and its related interfaces define the contract for executing
assistant-related actions in the system. These interfaces are crucial for
implementing various modes of interaction with the assistant, such as text-based
chat and voice communication.

AssistantMessageExecutor handles text-based chat interactions. It defines a Chat
method that processes messaging requests and returns any errors encountered during
the chat process.

AssistantTalkExecutor is responsible for voice-based interactions. Its Talk method
takes care of processing talking requests and handles any errors that may occur
during the voice interaction.

AssistantExecutor combines both text and voice capabilities, allowing for a more
versatile assistant that can handle multiple modes of communication. By embedding
both AssistantMessageExecutor and AssistantTalkExecutor, it ensures that any
implementing type can handle both chat and talk functionalities.

These interfaces provide a clean separation of concerns and allow for easy
extension of the assistant's capabilities in the future. They also promote
loose coupling between the assistant's implementation and the rest of the system,
making it easier to maintain and evolve the codebase over time.
*/

type AssistantExecutor interface {

	// init after creation to intilize all fields
	Init(
		ctx context.Context,
		communication internal_adapter_requests.Communication,
	) error

	// called when connection established
	Connect(
		ctx context.Context,
		assistantId uint64,
		assistantConversationId uint64,
	) error

	// name
	Name() string

	// conversation
	Talk(
		ctx context.Context,
		messageid string,
		msg *types.Message,
		communcation internal_adapter_requests.Communication,
	) error

	// disconnect
	Disconnect(
		ctx context.Context,
		assistantId uint64,
		assistantConversationId uint64,
	) error
}

/**
 * ToolExecutor is an interface that defines methods for executing tools and retrieving function definitions.
 *
 * This interface provides a contract for implementing tool execution functionality,
 * allowing for flexible and extensible tool management within the system.
 */

type ToolExecutor interface {

	// init tool executor
	//  get all the tools that is required for the assistant and intialize or do the dirty work that
	// optimize the execution or etc
	Init(
		ctx context.Context,
		communication internal_adapter_requests.Communication,
	) error
	/**
	 * GetFunctionDefinitions retrieves function definitions based on the provided communication.
	 *
	 * This method is responsible for returning a slice of FunctionDefinition pointers
	 * that represent the available functions or tools based on the given communication context.
	 *
	 * @param com The communication object containing context for function definition retrieval.
	 * @return A slice of FunctionDefinition pointers representing available functions or tools.
	 */
	GetFunctionDefinitions() []*lexatic_backend.FunctionDefinition

	/**
	 * Execute performs the execution of a tool call using the provided communication.
	 *
	 * This method is responsible for executing the specified tool call and returning
	 * the result as a Content object. If an error occurs during execution, it should be returned.
	 *
	 * @param call The ToolCall object containing information about the tool to be executed.
	 * @param com The communication object containing context for the execution.
	 * @return A pointer to a Content object representing the execution result, and an error if any occurred.
	 */
	Execute(
		ctx context.Context,
		messageid string,
		call *lexatic_backend.ToolCall,
		communication internal_adapter_requests.Communication,
	) *types.Content

	/**
	* ExecuteAll executes multiple tool calls and returns the results
	*
	* Parameters:
	*   - calls: A slice of ToolCall pointers to be executed
	*   - com: A Communication object for handling requests
	*
	* Returns:
	*   - A slice of Content pointers containing the results of the tool calls
	*   - An error if any occurred during execution
	 */
	ExecuteAll(
		ctx context.Context,
		messageid string,
		calls []*lexatic_backend.ToolCall,
		communication internal_adapter_requests.Communication) []*types.Content
}
