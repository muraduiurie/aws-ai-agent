# aws-ai-agent

A Go-based **MCP (Model Context Protocol) server** that exposes AWS and Kubernetes resource management as tools consumable by AI agents (Claude Desktop, Cursor, and any MCP-compatible host).

The server runs as a subprocess communicating over stdio. The host application pipes stdin/stdout for JSON-RPC; no network port is opened.

---

## Architecture

```
cmd/server/main.go        ← Entry point: loads config, wires AWS factory, ClientHolder, MCP server
internal/aws/             ← AWS client factory + one file per service (ec2.go, eks.go)
internal/kube/            ← Kubernetes client interface, pod operations, ClientHolder (holder.go)
internal/mcp/             ← MCP server struct, tool registration, stdio transport
internal/tools/           ← Tool definitions, handlers, and registration per service
pkg/agent/                ← Agent interface + Claude implementation
pkg/config/               ← Unified config: YAML via Viper (AI) + env vars (AWS)
pkg/types/                ← Shared output structs returned by tools
```

**Module:** `github.com/muraduiurie/aws-ai-agent`
**Go version:** 1.25

**Direct dependencies:**

| Package | Purpose |
|---|---|
| `github.com/mark3labs/mcp-go` | MCP protocol — server, tool registration, stdio transport |
| `github.com/aws/aws-sdk-go-v2` | AWS SDK core + config + EC2/EKS/STS clients |
| `github.com/anthropics/anthropic-sdk-go` | Claude API client |
| `github.com/spf13/viper` | YAML config file loading |
| `k8s.io/client-go` + `k8s.io/api` | Kubernetes client |

---

## Configuration

### Config file (AI settings)

The server reads a YAML file whose path is controlled by the `CONFIG_PATH` environment variable (default: `/etc/aws-ai-agent/config.yaml`). See `config.example.yaml` at the repo root.

```yaml
ai:
  claude:
    token: "your-anthropic-api-key"
    maxToken: 4096
    model: "claude-opus-4-6"
```

### Environment variables (AWS settings)

AWS configuration is read from environment variables at startup — no YAML entry needed:

| Variable | Default | Description |
|---|---|---|
| `AWS_REGION` | `eu-west-3` | Target AWS region (falls back to `AWS_DEFAULT_REGION`) |
| `AWS_PROFILE` | *(none)* | AWS shared config profile |
| `READ_ONLY` | `false` | Guard flag for future mutating tools (not yet enforced) |
| `CONFIG_PATH` | `/etc/aws-ai-agent/config.yaml` | Path to the YAML config file |

---

## Available tools

### EC2

| Tool | Description |
|---|---|
| `list_ec2_instances` | List instances in a region, optional state filter |

### EKS

| Tool | Description |
|---|---|
| `list_eks_clusters` | List cluster names in a region |
| `get_eks_cluster` | Get details of a specific EKS cluster |
| `get_eks_kubeconfig` | Generate a kubeconfig string for an EKS cluster |

### Kubernetes

| Tool | Description |
|---|---|
| `set_kubeconfig` | Inject a kubeconfig string to initialize the Kubernetes client at runtime |
| `list_pods` | List pods in a namespace, optional label selector |
| `get_pod` | Get details of a specific pod |
| `create_pod` | Create a pod from a JSON manifest string |
| `update_pod` | Update a pod from a JSON manifest string |
| `delete_pod` | Delete a pod by name |
| `get_pod_logs` | Fetch container logs, optional `tail_lines` |

### Kubernetes workflow

The Kubernetes client is **not initialized at startup**. The intended agent workflow is:

```
get_eks_kubeconfig  →  set_kubeconfig  →  any Kubernetes tools
```

`set_kubeconfig` accepts the kubeconfig YAML string returned by `get_eks_kubeconfig` and atomically initializes the client. All other Kubernetes tools return an error if called before `set_kubeconfig`.

---

## Agent package

`pkg/agent` provides the AI agent abstraction used to drive agentic logic on top of the MCP tools.

```go
cfg, err := config.Load()
agent, err := agent.NewAgent(&cfg.AI.Claude)
response, err := agent.Ping(ctx) // verifies API key and connectivity
```

`Ping` sends a minimal 16-token message to the model. A successful response confirms the API key is valid and the Claude endpoint is reachable.

---

## Development

### Commands

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

# Lint (requires golangci-lint)
golangci-lint run
```

### Adding a new tool

Every tool is three parts in the same file under `internal/tools/<service>.go`:

1. **Definition** — `mcp.NewTool(...)` with name, description, and input schema
2. **Handler** — a `server.ToolHandlerFunc` closure over the relevant client/factory
3. **Registration** — a `Register<Service>(s, ...)` function

Call the registration function from `registerTools()` in `internal/mcp/server.go`.

### Adding a new AWS service

1. `go get github.com/aws/aws-sdk-go-v2/service/<name>`
2. Add `internal/aws/<service>.go` with methods on `Factory`
3. Add `internal/tools/<service>.go` with definition, handler, and registration
4. Add output struct(s) to `pkg/types/<service>.go`
5. Call `Register<Service>` from `registerTools()` in `internal/mcp/server.go`

### Key design rules

- `internal/tools/` files must not import each other; shared types go in `pkg/types/`
- AWS service clients are created per-call with a region override inside `Factory` methods — never in `internal/tools/`
- The Kubernetes `Client` is an interface (`internal/kube/Client`); the concrete type is unexported — use `NewClient(*string)` for mocking in tests
- The Kubernetes client is lazily initialized via `kube.ClientHolder`; all kube handlers call `holder.Get()` and error if unset
- `pkg/config.Load()` is the single config entry point — reads YAML and env vars together; no global state

### mcp-go API quick reference

```go
// Tool definition
mcp.WithDescription("...")       // tool-level description
mcp.Description("...")           // property-level description (different function)
mcp.Required()                   // mark a property required

// Handler argument extraction
req.RequireString("key")         // required; returns error if missing
req.GetString("key", "default")  // optional with default
req.GetInt("key", 0)             // optional int

// Handler return values
mcp.NewToolResultJSON(value)     // structured JSON result
mcp.NewToolResultText("...")     // plain text result
mcp.NewToolResultError("msg")    // error visible to the agent
```
