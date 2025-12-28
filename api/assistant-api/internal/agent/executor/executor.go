// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_executor

import (
	"context"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters"
	"github.com/rapidaai/pkg/types"
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
	Initialize(
		ctx context.Context,
		communication internal_adapter_requests.Communication,
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
	Close(
		ctx context.Context,
		communcation internal_adapter_requests.Communication,
	) error
}
