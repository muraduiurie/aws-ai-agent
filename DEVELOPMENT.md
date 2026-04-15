# Development Notes

## Phase 1 — Project scaffolding

We started by establishing the folder structure before writing any logic. The layout separates concerns clearly: `cmd/server` for the entry point, `internal/` for all private packages (`mcp`, `tools`, `aws`, `config`), and `pkg/types` for shared output structs. Everything was placed directly at the repo root since the repo itself is the project.

A `CLAUDE.md` was created early to capture the module name, build commands, architecture overview, and design rules — intended to give future Claude Code sessions immediate context without needing to re-read the whole codebase.

## Phase 2 — Dependencies

Four direct dependencies were installed at this phase:

- `github.com/mark3labs/mcp-go` — MCP protocol implementation (server, tool registration, stdio transport)
- `github.com/aws/aws-sdk-go-v2` — AWS SDK core
- `github.com/aws/aws-sdk-go-v2/config` — credential and region resolution
- `github.com/aws/aws-sdk-go-v2/service/ec2` — EC2 service client

Two further direct dependencies were added in later phases: `github.com/anthropics/anthropic-sdk-go` (Phase 9) and `github.com/spf13/viper` (Phase 10a). The Kubernetes client libraries (`k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go`) were also pulled in alongside the kube tooling.

## Phase 3 — Wiring the entry point

Three supporting packages were scaffolded before writing `main.go`:

- `internal/config` — reads `AWS_REGION`, `AWS_PROFILE`, and `READ_ONLY` from environment variables at startup *(later removed in Phase 10b; logic merged into `pkg/config`)*
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

## Phase 8 — Lazy Kubernetes client initialization

The Kubernetes client was originally constructed at startup from the `KUBECONFIG_PATH` environment variable. This was replaced with a **lazy-init** model so the AI agent can inject the kubeconfig at runtime after retrieving it from EKS.

Key changes:

- **`internal/kube/holder.go`** — new `ClientHolder` struct (thread-safe via `sync.RWMutex`) with two methods:
  - `Set(kubeconfig string) error` — parses the kubeconfig and atomically replaces the held client; on error the previous client (if any) is preserved
  - `Get() (Client, bool)` — returns the current client and whether it has been initialized
- **`cmd/server/main.go`** — kubeconfig loading and `KUBECONFIG_PATH` env var removed; a zero-value `&kube.ClientHolder{}` is passed to the MCP server instead of a live client
- **`internal/mcp/server.go`** — `KubeClient kube.Client` field replaced with `KubeHolder *kube.ClientHolder`
- **`internal/tools/kube.go`** — `RegisterKube` now takes `*kube.ClientHolder`; all six kube handlers call `holder.Get()` at the top and return a tool error if the client is not yet set; a new `set_kubeconfig` tool was added
- **`set_kubeconfig` tool** — accepts a kubeconfig YAML string and calls `holder.Set`; intended to be called by the agent immediately after `get_eks_kubeconfig`

The intended agent workflow is: `get_eks_kubeconfig` → `set_kubeconfig` → any Kubernetes tools.

Tests added in `internal/kube/holder_test.go` and `internal/tools/kube_connection_test.go` cover: uninitialized Get, invalid kubeconfig rejection, single-cluster connect, switching between clusters, reconnecting to a previous cluster, failed-Set preserving the existing client, and the not-initialized guard on kube tool handlers. Handlers are tested by calling the unexported handler functions directly (the test file uses `package tools`).

## Phase 9 — Agent package

A new `pkg/agent` package introduces the AI agent abstraction:

- **`pkg/agent/agent.go`** — `Agent` interface with a single method: `Ping(ctx context.Context) (string, error)`
- **`pkg/agent/claude.go`** — `claudeAgent` concrete type backed by the Anthropic Go SDK (`github.com/anthropics/anthropic-sdk-go`); `NewAgent(cfg *config.ClaudeConfig) (Agent, error)` is the constructor — updated in Phase 10a to take config instead of raw strings

`Ping` sends a single 16-token "ping" message to the model and returns the response text. It is the minimal connectivity check: a successful call proves the API key is valid and the model is reachable.

Default model: `claude-opus-4-6`.

## Phase 10 — Application config package and agent wiring

### Phase 10a — pkg/config and agent wiring

A new `pkg/config` package was introduced to centralise runtime configuration via a YAML file read by Viper (`github.com/spf13/viper`).

Key decisions:

- **File location via env var** — `CONFIG_PATH` controls the path; falls back to `/etc/aws-ai-agent/config.yaml` so Kubernetes deployments can mount a `ConfigMap` or `Secret` at the well-known path without any extra configuration
- **Struct hierarchy** — `Config → AIConfig → ClaudeConfig`; the nesting leaves room for adding other AI providers or top-level sections without breaking the existing layout
- **camelCase YAML keys** — `token`, `maxToken`, `model`; Viper stores keys lowercase internally but the `mapstructure` tags map them correctly to the exported struct fields
- **`config.example.yaml`** at the repo root is the canonical reference for the expected file format

`NewAgent` was updated to accept `*config.ClaudeConfig` instead of raw `apiKey, model string` parameters. The struct's three fields map directly onto the agent:
  - `Token` → Anthropic API key
  - `Model` → model ID passed to every API call
  - `MaxToken` → stored on the `claudeAgent` struct for use in future non-Ping calls (`Ping` keeps its own 16-token cap since it is a connectivity check, not a real task)

### Phase 10b — Consolidate internal/config into pkg/config

`internal/config` was deleted and its logic merged into `pkg/config`:

- `AWSConfig` struct added to `pkg/config` with `Profile`, `Region`, and `ReadOnly` fields
- `loadAWSFromEnv()` private helper carries the env-var resolution logic verbatim from the old package (`AWS_REGION` → `AWS_DEFAULT_REGION` → `"eu-west-3"` fallback chain)
- `Config.AWS` is tagged `mapstructure:"-"` so Viper does not try to populate it from the YAML file; `Load()` calls `loadAWSFromEnv()` after unmarshaling and attaches the result
- `internal/aws/client.go` now imports `pkg/config` and `NewFactory` accepts `*config.AWSConfig` (instead of `*config.Config`), limiting its surface to only the fields it actually uses
- `cmd/server/main.go` updated: `config.Load()` now returns `(*Config, error)`, so the startup sequence gains a fatal error check for config loading failure
