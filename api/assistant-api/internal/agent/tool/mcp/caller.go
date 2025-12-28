// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_agent_mcp_tool

import internal_agent_local_tool "github.com/rapidaai/api/assistant-api/internal/agent/tool/local"

// Just a placeholder for MCP specific tool caller interface
type MCPCaller interface {

	// name
	Name() string

	// list of tool callers will be returned
	Tools() ([]internal_agent_local_tool.ToolCaller, error)
}
