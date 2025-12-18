package internal_agent_mcp_tool

import internal_agent_local_tool "github.com/rapidaai/api/assistant-api/internal/agent/tool/local"

// Just a placeholder for MCP specific tool caller interface
type MCPCaller interface {

	// name
	Name() string

	// list of tool callers will be returned
	Tools() ([]internal_agent_local_tool.ToolCaller, error)
}
