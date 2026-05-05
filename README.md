# gopen-manus

Go port of OpenManus core logic (agents, planning flow, schema), with MCP client support wired via the official MCP Go SDK to connect to OpenManus MCP tools.

## Quick Start

```bash
export OPENAI_API_KEY=sk-...
go run ./cmd/manus -prompt "Plan my day"
# Or run the full planning flow:
go run ./cmd/run-flow
```

As long as `OPENAI_API_KEY` is set, `ManusAgent` automatically boots with the OpenAI client; if it is missing, the agent falls back to the built-in `NoopLLM` so you can develop offline.

### Optional environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `OPENAI_MODEL` | `gpt-4o` | Model name to use |
| `OPENAI_MAX_TOKENS` | `2048` | Max tokens in each completion |
| `OPENAI_MAX_INPUT_TOKENS` | `12000` | Max tokens accepted as input (0 disables the limit) |
| `OPENAI_TEMPERATURE` | `0.7` | Sampling temperature |
| `OPENAI_API_BASE_URL` | empty | Preferred OpenAI-compatible base URL (e.g., `https://api.longcat.chat/openai`) |
| `OPENAI_BASE_URL` | empty | Fallback OpenAI API base (legacy name, used if `OPENAI_API_BASE_URL` is unset) |
| `OPENMANUS_WORKSPACE` | current working directory | Directory baked into Manus system prompt |

Timeouts and other defaults live in `internal/config/config.go`. Adjust `config.Settings.LLM.RequestTimeout`
(a `time.Duration`) there if you need to cap network calls when using OpenAI-compatible providers.

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
