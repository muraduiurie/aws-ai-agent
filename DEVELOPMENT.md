# Development Notes

## Phase 1 — Project scaffolding

We started by establishing the folder structure before writing any logic. The layout separates concerns clearly: `cmd/server` for the entry point, `internal/` for all private packages (`mcp`, `tools`, `aws`, `config`), and `pkg/types` for shared output structs. Everything was placed directly at the repo root since the repo itself is the project.

A `CLAUDE.md` was created early to capture the module name, build commands, architecture overview, and design rules — intended to give future Claude Code sessions immediate context without needing to re-read the whole codebase.

## Phase 2 — Dependencies

Four direct dependencies were installed:

- `github.com/mark3labs/mcp-go` — MCP protocol implementation (server, tool registration, stdio transport)
- `github.com/aws/aws-sdk-go-v2` — AWS SDK core
- `github.com/aws/aws-sdk-go-v2/config` — credential and region resolution
- `github.com/aws/aws-sdk-go-v2/service/ec2` — EC2 service client

## Phase 3 — Wiring the entry point

Three supporting packages were scaffolded before writing `main.go`:

- `internal/config` — reads `AWS_REGION`, `AWS_PROFILE`, and `READ_ONLY` from environment variables at startup
- `internal/aws` — `Factory` struct that calls `LoadDefaultConfig` once and holds the resolved `aws.Config`; individual service methods create per-call clients with region overrides
- `internal/mcp` — `Server` struct wrapping `*server.MCPServer` and `*aws.Factory`; exposes `registerTools()` and `Start()`

`main.go` is intentionally thin: load config → build factory → build MCP server → start. All transport and tool wiring is delegated to `internal/mcp`.

## Phase 4 — Transport

The MCP server uses **stdio** as its transport layer. `Start()` wraps the core MCP server in a `StdioServer` and calls `Listen(ctx, os.Stdin, os.Stdout)`. This is the standard transport for MCP servers launched as subprocesses by a host application (Claude Desktop, Cursor, etc.) — JSON-RPC messages flow over the process's stdin/stdout pipes. No network port is opened.

## Phase 5 — First tool: list_ec2_instances

Tools follow a three-part pattern established here and intended to be reused for every future tool:

1. **Definition** — `mcp.NewTool(...)` declares the tool name, description, and input schema. `region` is a required string parameter; `state` is optional.
2. **Handler** — a closure over `*aws.Factory` that extracts arguments via `req.RequireString` / `req.GetString`, calls the factory method, and returns `mcp.NewToolResultJSON`.
3. **Registration** — a `RegisterEC2(s, factory)` function called from `registerTools()` in the MCP server.

## Phase 6 — AWS SDK implementation

`ListEC2Instances` was added as a method on `Factory` in `internal/aws/ec2.go`. Key decisions:

- The EC2 client is created per-call with a region override so the tool argument controls the target region, not the factory default
- `ec2.NewDescribeInstancesPaginator` handles pagination automatically, collecting all instances across pages
- A private `toEC2Instance` helper converts the raw SDK type to `pkg/types/EC2Instance`, which has clean JSON tags and only exposes fields relevant to the AI agent

## Phase 7 — Review and cleanup

A full code review traced the interactions across all packages and identified two issues:

- `Factory.EC2()` in `client.go` was dead code — `ec2.go` creates its own clients inline. It was removed.
- `config.Config.ReadOnly` is loaded and logged but never enforced. This is an open gap: before any mutating tool is added, enforcement must be implemented in `registerTools()` or at the handler level.

`CLAUDE.md` was updated to reflect the actual client construction pattern and to document the `ReadOnly` gap explicitly.
