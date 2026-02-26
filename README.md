# gopen-manus

Go port of OpenManus core logic (agents, planning flow, schema), with MCP client support wired via the official MCP Go SDK to connect to OpenManus MCP tools.

## Entrypoints
- `cmd/manus`: mirrors `main.py`
- `cmd/run-flow`: mirrors `run_flow.py`
- `cmd/run-mcp`: mirrors `run_mcp.py` (stdio/sse via MCP Go SDK)
- `cmd/run-mcp-server`: convenience wrapper to launch the Python MCP server

## Structure
- `internal/schema`: core types (Message, ToolCall, Memory, AgentState)
- `internal/agent`: base agent loop, ReAct agent, ToolCall agent, MCP agent
- `internal/mcp`: MCP client and tool registry (SDK-backed)
- `internal/planning`: in-memory plan store
- `internal/flow`: base flow, planning flow, factory

## Notes
- LLM is an interface in `internal/llm`. A `NoopLLM` is provided as a default stub.
- MCP client uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk`).
