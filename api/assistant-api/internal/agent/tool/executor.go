// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_tool

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

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
	Initialize(
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
	GetFunctionDefinitions() []*protos.FunctionDefinition

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
		call *protos.ToolCall,
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
		calls []*protos.ToolCall,
		communication internal_adapter_requests.Communication) []*types.Content
}
