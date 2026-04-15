# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go-based MCP (Model Context Protocol) server that exposes AWS and Kubernetes resource management as tools consumable by AI agents.

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
cmd/server/main.go        ← Entry point: wires config, AWS factory, ClientHolder, MCP server, tool registration
internal/aws/             ← AWS client factory + one file per service (ec2.go, eks.go)
internal/kube/            ← Kubernetes client interface, pod operations, and ClientHolder (holder.go)
internal/mcp/             ← MCP server struct, tool registration, stdio transport
internal/tools/           ← One file per service (ec2.go, eks.go, kube.go); definitions, handlers, registration
pkg/agent/                ← Agent interface + Claude implementation (agent.go, claude.go)
pkg/config/               ← Unified config loader: YAML via Viper (AI) + env vars (AWS)
pkg/types/                ← Shared output structs (ec2.go, eks.go, kube.go)
```

## Implemented tools

| Tool | Package | Description |
|---|---|---|
| `list_ec2_instances` | EC2 | List instances in a region, optional state filter |
| `list_eks_clusters` | EKS | List cluster names in a region |
| `get_eks_cluster` | EKS | Get details of a specific EKS cluster |
| `get_eks_kubeconfig` | EKS | Generate a kubeconfig string for an EKS cluster |
| `set_kubeconfig` | Kube | Inject a kubeconfig string to initialize the Kubernetes client at runtime |
| `list_pods` | Kube | List pods in a namespace, optional label selector |
| `get_pod` | Kube | Get details of a specific pod |
| `create_pod` | Kube | Create a pod from a JSON manifest string |
| `update_pod` | Kube | Update a pod from a JSON manifest string |
| `delete_pod` | Kube | Delete a pod by name |
| `get_pod_logs` | Kube | Fetch container logs, optional tail_lines |

## Key design rules

- `internal/tools/` files must not import each other; shared types go in `pkg/types/`.
- AWS service clients are never constructed in `internal/tools/`. Each method on `Factory` creates its own client with a per-call region override (`eks.NewFromConfig(f.cfg, func(o *eks.Options) { o.Region = region })`). Tools receive the `Factory` and call its methods.
- The Kubernetes `Client` is an **interface** (`internal/kube/Client`). The concrete type (`client`) is unexported. `NewClient(*string)` returns the interface — use this for mocking in tests.
- The Kubernetes client is **lazily initialized** via `kube.ClientHolder`. The MCP server holds a `*kube.ClientHolder` (never a bare `kube.Client`). All kube tool handlers call `holder.Get()` at the top and return a tool error if the client is not yet set.
- Config is loaded once at startup in `main.go` and passed down; no global state. `pkg/config.Load()` is the single entry point — it reads the YAML file and env vars together.
- `config.AWSConfig.ReadOnly` is loaded but not yet enforced. Before adding any mutating tool (start/stop/terminate), add enforcement in `registerTools()` or at the handler level.

## Transport

The server uses **stdio** as its transport. `Start()` wraps the MCP server in `server.NewStdioServer` and calls `Listen(ctx, os.Stdin, os.Stdout)`. This is intentional — the binary is meant to be launched as a subprocess by a host (Claude Desktop, Cursor, etc.) which pipes stdin/stdout for JSON-RPC communication. Do not switch to SSE/HTTP unless explicitly asked.

## Kubernetes client connection

The Kubernetes client is **not initialized at startup**. `main.go` passes a zero-value `&kube.ClientHolder{}` to the MCP server. The AI agent must call `set_kubeconfig` before any other Kubernetes tool.

`kube.ClientHolder` (`internal/kube/holder.go`) is a thread-safe wrapper:
- `Set(kubeconfig string) error` — parses the kubeconfig content and atomically replaces the held client. On error the existing client (if any) is preserved.
- `Get() (Client, bool)` — returns the current client and whether it has been initialized.

`kube.NewClient(kubeconfig *string)`:
- `nil` → in-cluster config (`rest.InClusterConfig`)
- non-nil → parses the provided kubeconfig **content** (not a file path) via `clientcmd.RESTConfigFromKubeConfig`

The intended agent workflow is: `get_eks_kubeconfig` → `set_kubeconfig` → any Kubernetes tools.

## EKS kubeconfig generation

`Factory.GetEKSKubeconfig(ctx, region, name)` generates a ready-to-use kubeconfig string:
1. Fetches cluster endpoint and CA data via `DescribeCluster`
2. Generates a bearer token by presigning an STS `GetCallerIdentity` request with the `x-k8s-aws-id: <cluster-name>` header injected via a smithy build middleware — same mechanism as `aws eks get-token`
3. Token format: `k8s-aws-v1.<base64url(presigned_url)>`
4. Renders and returns a complete kubeconfig YAML string

The returned string should be passed to the `set_kubeconfig` tool to initialize the Kubernetes client.

## Adding a new tool

Every tool consists of three parts, all in the same file under `internal/tools/<service>.go`:
1. **Definition** — `mcp.NewTool(...)` with name, description, and input schema
2. **Handler** — a `server.ToolHandlerFunc` closure over the relevant client/factory
3. **Registration** — a `Register<Service>(...)` function

Call the registration function from `registerTools()` in `internal/mcp/server.go` — that is the single place all tools are wired in.

## mcp-go API notes

- Tool-level description: `mcp.WithDescription("...")`
- Property-level description: `mcp.Description("...")` — different function, easy to mix up
- Required property: `mcp.Required()`
- Extract required argument in handler: `req.RequireString("key")` — returns an error if missing
- Extract optional argument in handler: `req.GetString("key", "default")`
- Extract optional int: `req.GetInt("key", 0)`
- Return structured data: `mcp.NewToolResultJSON(value)` — returns `(*CallToolResult, error)`
- Return plain text: `mcp.NewToolResultText("...")`
- Return error to the agent: `mcp.NewToolResultError("msg")` or `mcp.NewToolResultErrorFromErr("msg", err)`

## Application config (pkg/config)

`pkg/config` loads the application YAML configuration file using Viper.

**Config file location**: read from the `CONFIG_PATH` environment variable; defaults to `/etc/aws-ai-agent/config.yaml`.

**Structure** (`config.example.yaml` at the repo root is the canonical reference):

```yaml
ai:
  claude:
    token: "your-anthropic-api-key"
    maxToken: 4096
    model: "claude-opus-4-6"
```

AWS configuration is **not** in the YAML file — it is read from environment variables as before (`AWS_REGION` / `AWS_DEFAULT_REGION`, `AWS_PROFILE`, `READ_ONLY`).

**Go structs**:

```
Config
├── AI (AIConfig)          ← from YAML via Viper
│   └── Claude (ClaudeConfig)
│       ├── Token    string
│       ├── MaxToken int
│       └── Model    string
└── AWS (AWSConfig)        ← from environment variables
    ├── Profile  string
    ├── Region   string
    └── ReadOnly bool
```

**Usage**:

```go
cfg, err := config.Load()
// cfg.AI.Claude.{Token,Model,MaxToken}
// cfg.AWS.{Region,Profile,ReadOnly}
```

Note: Viper normalizes YAML keys case-insensitively internally; the `mapstructure` tags preserve camelCase mapping (`maxToken` → `MaxToken`). `AWSConfig` carries `mapstructure:"-"` so Viper ignores it during YAML unmarshaling.

## Agent package

`pkg/agent` provides the AI agent abstraction used to drive agentic logic on top of the MCP tools.

- **`Agent` interface** (`agent.go`) — currently exposes `Ping(ctx context.Context) (string, error)`
- **`NewAgent(cfg *config.ClaudeConfig) (Agent, error)`** (`claude.go`) — constructs a `claudeAgent` backed by the Anthropic Go SDK; reads `Token`, `Model`, and `MaxToken` from the config struct
- **`claudeAgent.Ping`** — sends a 16-token "ping" message and returns the model's response text; use it to verify API key validity and connectivity before running agent workflows

Default model: `claude-opus-4-6`.

```go
cfg, err := config.Load()
agent, err := agent.NewAgent(&cfg.AI.Claude)
response, err := agent.Ping(ctx)
```

## Adding a new AWS service

1. `go get github.com/aws/aws-sdk-go-v2/service/<name>`
2. Add `internal/aws/<service>.go` with methods on `Factory`
3. Add `internal/tools/<service>.go` with definition, handler, and registration
4. Add output struct(s) to `pkg/types/<service>.go`
5. Call `Register<Service>` from `registerTools()` in `internal/mcp/server.go`
