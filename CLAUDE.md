# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go-based MCP (Model Context Protocol) server that exposes AWS resource management as tools consumable by AI agents.

Module: `github.com/muraduiurie/aws-ai-agent`
Go version: 1.24

## Commands

```bash
# Build
go build ./...

# Run server
go run ./cmd/server

# Test all
go test ./...

# Test a single package
go test ./internal/tools/...

# Test a single test
go test ./internal/tools/... -run TestFunctionName

# Lint (assumes golangci-lint installed)
golangci-lint run
```

## Architecture

```
cmd/server/main.go        ← Entry point: wires config, AWS client factory, MCP server, and tool registration
internal/config/          ← Server config (read-only mode, allowed regions, etc.)
internal/aws/             ← AWS client factory: handles credential loading and region logic
internal/mcp/             ← MCP server setup and tool registration
internal/tools/           ← One file per AWS service (ec2.go, s3.go, …); each file registers its tools with the MCP server
pkg/types/                ← Shared input/output structs used across tools
```

### Key design rules
- `internal/tools/` files must not import each other; shared types go in `pkg/types/`.
- AWS service clients are never constructed in `internal/tools/`. Each method on `Factory` creates its own client with a per-call region override (`ec2.NewFromConfig(f.cfg, func(o *ec2.Options) { o.Region = region })`). Tools receive the `Factory` and call its methods.
- Config is loaded once at startup in `main.go` and passed down; no global state.
- `config.Config.ReadOnly` is loaded but not yet enforced. Before adding any mutating tool (start/stop/terminate), add enforcement in `registerTools()` or at the handler level.

### Transport
The server uses **stdio** as its transport. `Start()` wraps the MCP server in `server.NewStdioServer` and calls `Listen(ctx, os.Stdin, os.Stdout)`. This is intentional — the binary is meant to be launched as a subprocess by a host (Claude Desktop, Cursor, etc.) which pipes stdin/stdout for JSON-RPC communication. Do not switch to SSE/HTTP unless explicitly asked.

### Adding a new tool
Every tool consists of three parts, all in the same file under `internal/tools/<service>.go`:
1. **Definition** — `mcp.NewTool(...)` with name, description, and input schema
2. **Handler** — a `server.ToolHandlerFunc` closure over `*aws.Factory`
3. **Registration** — a `Register<Service>(s *server.MCPServer, factory *aws.Factory)` function

Once the file is created, call the registration function from `registerTools()` in `internal/mcp/server.go`. That is the single place all tools are wired in.

### mcp-go API notes
- Tool-level description: `mcp.WithDescription("...")`
- Property-level description: `mcp.Description("...")` — different function, easy to mix up
- Required property: `mcp.Required()`
- Extract required argument in handler: `req.RequireString("key")` — returns an error if missing
- Extract optional argument in handler: `req.GetString("key", "default")`
- Return structured data: `mcp.NewToolResultJSON(value)` — returns `(*CallToolResult, error)`
- Return error to the agent: `mcp.NewToolResultError("msg")` or `mcp.NewToolResultErrorFromErr("msg", err)`

### Adding a new AWS service
1. Add the SDK package to `go.mod` via `go get github.com/aws/aws-sdk-go-v2/service/<name>`
2. Add a new file `internal/aws/<service>.go` with methods on `Factory`
3. Add a new file `internal/tools/<service>.go` with definition, handler, and registration
4. Add the output struct(s) to `pkg/types/<service>.go`
5. Call `Register<Service>` from `registerTools()` in `internal/mcp/server.go`
