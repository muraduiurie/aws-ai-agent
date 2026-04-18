# PROMPT.md — Development Reproduction Guide

This file contains prompts and verification checklists for reproducing the development of this project phase by phase. Each prompt is designed for the Claude Code environment and uses structured prompting techniques to maximize precision and minimize ambiguity.

---

## 1. MCP Server

### Phase 1 — Project Scaffolding

#### Why

Before any logic can be written, the project needs a clear, agreed-upon structure. Establishing the directory layout and `CLAUDE.md` first means every subsequent prompt can reference concrete file paths and package names without ambiguity. Skipping this step leads to inconsistent placement of code and forces repeated restructuring later.

#### Prompt

```
You are a Go software architect. Your task is to scaffold a new Go project from scratch — do not write any logic yet, only establish the directory structure, the Go module, and a minimal CLAUDE.md file.

<context>
Project: A Go-based MCP (Model Context Protocol) server that will expose AWS and Kubernetes resource management as tools consumable by AI agents (Claude Desktop, Cursor, and other MCP-compatible hosts).
Module name: github.com/muraduiurie/aws-ai-agent
Go version: 1.25
</context>

<requirements>
Create the following directory layout (empty files with a minimal package declaration are sufficient):

  cmd/server/main.go          ← entry point
  internal/aws/               ← AWS client factory (one file per service later)
  internal/mcp/               ← MCP server struct and transport
  internal/tools/             ← tool definitions, handlers, and registration (one file per service)
  pkg/types/                  ← shared output structs returned by tools
  pkg/config/                 ← unified configuration package

Constraints:
- Run `go mod init github.com/muraduiurie/aws-ai-agent` to initialize the module.
- Do NOT add any dependencies yet.
- Do NOT write any logic in main.go — a bare `func main() {}` is fine.
- Each directory needs at least one .go file with a correct package declaration.
</requirements>

<output>
After creating the files, create a CLAUDE.md at the repo root containing:
1. Project description (one sentence)
2. Module name and Go version
3. Build, run, test, and lint commands
4. Architecture diagram (ASCII tree matching the directory layout above)
5. A "Key design rules" section with at least these rules:
   - internal/tools/ files must not import each other; shared types go in pkg/types/
   - No global state; config is loaded once in main.go and passed down
   - AWS service clients are created per-call with a region override inside Factory methods, never in internal/tools/
</output>
```

#### Verification Points

- `go build ./...` succeeds with no errors on an empty module.
- `go mod init` was called — `go.mod` exists with the correct module name and Go version.
- All directories listed in the requirements exist and contain at least one `.go` file with the right package name (e.g., `package main` for `cmd/server/`, `package aws` for `internal/aws/`).
- `CLAUDE.md` is present at the repo root and covers all five required sections.
- No logic, no imports, no dependencies — this phase is structure only.

---

### Phase 2 — Dependencies

#### Why

Dependency management is isolated into its own phase so that `go.mod` and `go.sum` reflect only intentional, direct imports — not whatever `go get` happens to pull in transitively. Doing this before writing any code prevents packages from ending up marked `// indirect` (which happened when deps were added mid-implementation) and keeps the module file clean as a source of truth.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). The directory structure is already in place. Your task is to install the four core dependencies needed for the MCP server and AWS integration.

<dependencies>
Install exactly these packages using `go get`:
1. github.com/mark3labs/mcp-go          — MCP protocol (server, tool registration, stdio transport)
2. github.com/aws/aws-sdk-go-v2         — AWS SDK core
3. github.com/aws/aws-sdk-go-v2/config  — credential and region resolution
4. github.com/aws/aws-sdk-go-v2/service/ec2 — EC2 service client
</dependencies>

<constraints>
- Run `go mod tidy` after installing to remove any phantom indirect entries.
- Do not write any code — this phase is dependency management only.
- Do not install any packages beyond the four listed above.
</constraints>

<output>
Confirm which packages now appear as `require` entries in go.mod and whether they are marked direct or indirect.
</output>
```

#### Verification Points

- `go.mod` lists all four packages as direct dependencies (no `// indirect` on them).
- `go.sum` was updated — it exists and is non-empty.
- `go build ./...` still succeeds.
- No extra packages were added beyond the four specified.

---

### Phase 3 — Entry Point and Supporting Packages

#### Why

The three core packages (`internal/aws`, `internal/mcp`, `cmd/server`) form the backbone that all tools plug into. Wiring them together — even with empty tool registration — validates that the compile graph is correct, the stdio transport starts cleanly, and the `Factory` → `Server` → `Start` chain works before any tool logic is introduced. It also enforces the thin-`main` rule early, before there is anything tempting to put there.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25) with the directory structure and dependencies already in place. Your task is to implement three supporting packages and a thin main.go. Do not implement any tools yet.

<package_1: internal/aws>
Create internal/aws/client.go containing:
- A `Factory` struct that holds an `aws.Config` (from aws-sdk-go-v2).
- A `NewFactory(ctx context.Context, region, profile string) (*Factory, error)` constructor that calls `config.LoadDefaultConfig` once with the provided region and profile.
- Individual service methods (e.g., for EC2, EKS) will be added in later files — do not implement them now.

Design rule: service clients are NEVER stored on Factory. Each service method creates its own client per call with a region override.
</package_1>

<package_2: internal/mcp>
Create internal/mcp/server.go containing:
- A `Server` struct wrapping `*server.MCPServer` from mcp-go and `*aws.Factory`.
- `NewServer(awsFactory *aws.Factory) *Server` constructor.
- A private `registerTools()` method (empty body for now — tools will be wired here later).
- A `Start(ctx context.Context) error` method that calls `registerTools()` then starts the MCP server.
  Transport: stdio only. Use `server.NewStdioServer` and `Listen(ctx, os.Stdin, os.Stdout)`.
</package_2>

<package_3: cmd/server/main.go>
Implement main.go so it:
1. Creates a context.
2. Reads AWS_REGION (default "eu-west-3"), AWS_PROFILE, and READ_ONLY env vars.
3. Calls aws.NewFactory with the resolved values.
4. Calls mcp.NewServer with the factory.
5. Calls s.Start(ctx) and exits fatally on error.

main.go must be thin — no business logic. All wiring belongs in the packages above.
</package_3>

<constraints>
- Do not implement any tools yet.
- Do not open any network ports — stdio transport only.
- All error paths must be handled (no ignored errors).
</constraints>
```

#### Verification Points

- `go build ./...` succeeds.
- `go run ./cmd/server` starts without panicking (it will block on stdin, which is correct).
- `internal/aws/client.go`: `Factory` holds `aws.Config`, not a service client. No EC2/EKS client stored on the struct.
- `internal/mcp/server.go`: `Start()` uses `server.NewStdioServer` → `Listen(ctx, os.Stdin, os.Stdout)`. No HTTP, no SSE.
- `main.go` is under ~40 lines and contains no business logic.
- All three env vars are read; `READ_ONLY` is logged at startup even though it is not yet enforced.

---

### Phase 4 — First Tool: `list_ec2_instances`

#### Why

The first tool does double duty: it delivers real functionality (EC2 instance listing) and simultaneously establishes the three-part pattern (definition / handler / registration) that every subsequent tool must follow. Getting this pattern right once — with pagination, per-call client construction, and clean output types — makes every later tool a repetition of a known-good template rather than a fresh design decision.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). The MCP server skeleton is in place. Your task is to implement the first tool end-to-end following the three-part pattern that will be used for every future tool.

<three_part_pattern>
Every tool in this project consists of exactly three parts, all in the same file under internal/tools/<service>.go:
1. Definition  — mcp.NewTool(...) with name, description, and input schema.
2. Handler     — a server.ToolHandlerFunc closure over the relevant client/factory.
3. Registration — a Register<Service>(s *server.MCPServer, ...) function called from registerTools().
</three_part_pattern>

<task>
Implement the `list_ec2_instances` tool.

Step 1 — Output type (pkg/types/ec2.go):
Create an EC2Instance struct with JSON tags. Include at minimum:
  InstanceID, InstanceType, State, PrivateIP, PublicIP, LaunchTime, Tags (map[string]string).
Do not expose every SDK field — only what is useful to an AI agent.

Step 2 — AWS method (internal/aws/ec2.go):
Add a `ListEC2Instances(ctx, region, state string) ([]types.EC2Instance, error)` method on Factory.
- Create the EC2 client per-call with a region override (never store it on Factory).
- Use ec2.NewDescribeInstancesPaginator to handle pagination.
- If state is non-empty, apply a Name=instance-state-name filter.
- Convert each reservation/instance with a private toEC2Instance helper.

Step 3 — Tool (internal/tools/ec2.go):
- Definition: name="list_ec2_instances", required string param "region", optional string param "state".
- Handler: extract args via req.RequireString / req.GetString, call factory.ListEC2Instances, return mcp.NewToolResultJSON on success, mcp.NewToolResultErrorFromErr on failure.
- Registration: RegisterEC2(s, factory) — adds the tool to the MCP server.

Step 4 — Wire it:
Call RegisterEC2 from registerTools() in internal/mcp/server.go.
</task>

<mcp_go_api_reference>
mcp.WithDescription("...")        // tool-level description
mcp.Description("...")            // property-level description (different function!)
mcp.Required()                    // mark property required
req.RequireString("key")          // required arg; returns error if absent
req.GetString("key", "default")   // optional arg with default
mcp.NewToolResultJSON(value)      // structured JSON result
mcp.NewToolResultErrorFromErr("msg", err) // error visible to the agent
</mcp_go_api_reference>

<constraints>
- The EC2 client must be created inside the method body, not stored on Factory.
- internal/tools/ec2.go must not import internal/aws directly for data types — use pkg/types.
- All error paths must return mcp.NewToolResultErrorFromErr, never panic or log.Fatal.
</constraints>
```

#### Verification Points

- `go build ./...` succeeds.
- `pkg/types/ec2.go` defines `EC2Instance` with clean JSON tags and no raw SDK types exposed.
- `internal/aws/ec2.go`: the EC2 client is created with `ec2.NewFromConfig(f.cfg, func(o *ec2.Options) { o.Region = region })` inside the method — not stored on `Factory`.
- Pagination: `ec2.NewDescribeInstancesPaginator` is used (not a single `DescribeInstances` call).
- `internal/tools/ec2.go` follows all three parts of the pattern in a single file.
- `registerTools()` in `internal/mcp/server.go` calls `tools.RegisterEC2(s, f.awsFactory)`.
- Running the server and calling `list_ec2_instances` via an MCP client returns JSON.

---

### Phase 5 — EKS Tools

#### Why

EKS tooling is the bridge between AWS and Kubernetes. `get_eks_kubeconfig` is particularly critical: it generates a bearer token via a presigned STS request (the same mechanism `aws eks get-token` uses), which is the only way to authenticate to an EKS cluster without pre-placed credentials. Without this phase, the Kubernetes tools in Phase 6 have no way to receive a valid kubeconfig from the agent at runtime.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). EC2 tooling is in place. Your task is to add three EKS tools and a kubeconfig generator following the same three-part pattern.

<dependencies>
Install these packages before writing any code:
  go get github.com/aws/aws-sdk-go-v2/service/eks
  go get github.com/aws/aws-sdk-go-v2/service/sts
  go get github.com/aws/smithy-go
</dependencies>

<output_types: pkg/types/eks.go>
Define:
- EKSCluster — fields: Name, Status, Endpoint, KubernetesVersion, RoleARN, Tags.
- EKSClusterList — a slice wrapper or just []string for cluster names.
</output_types>

<aws_methods: internal/aws/eks.go>
Add to Factory:
1. ListEKSClusters(ctx, region) ([]string, error)
   - Use eks.NewListClustersPaginator for pagination.

2. GetEKSCluster(ctx, region, name) (*types.EKSCluster, error)
   - Single DescribeCluster call.

3. GetEKSKubeconfig(ctx, region, name) (string, error)
   - Fetch cluster endpoint + CA data via DescribeCluster.
   - Generate a bearer token: presign an STS GetCallerIdentity request with the
     x-k8s-aws-id: <cluster-name> header injected via a smithy presign middleware.
   - Token format: k8s-aws-v1.<base64url(presigned_url)>  (URL-safe base64, no padding).
   - Render and return a complete kubeconfig YAML string. The kubeconfig must be
     usable directly with client-go without any modification.

Design rule: EKS and STS clients are created per-call with region overrides — never stored on Factory.
</aws_methods>

<tools: internal/tools/eks.go>
Implement three tools following the three-part pattern:
1. list_eks_clusters  — required: region
2. get_eks_cluster    — required: region, name
3. get_eks_kubeconfig — required: region, name
   (returns the kubeconfig YAML string as plain text, not JSON)

RegisterEKS(s, factory) wires all three.
</tools>

<constraints>
- Call RegisterEKS from registerTools() in internal/mcp/server.go.
- get_eks_kubeconfig must return mcp.NewToolResultText (not JSON) — the output is YAML.
- Do not store any client on Factory.
- The presigned STS URL must include the x-k8s-aws-id header — this is mandatory for EKS authentication.
</constraints>
```

#### Verification Points

- `go build ./...` succeeds.
- `internal/aws/eks.go`: EKS and STS clients are created inside each method body with `eks.NewFromConfig(..., regionOverride)` — not stored on `Factory`.
- `GetEKSKubeconfig`: token starts with `k8s-aws-v1.` and the base64 portion decodes to a valid STS presigned URL containing `x-k8s-aws-id` in the query string.
- The rendered kubeconfig YAML contains `certificate-authority-data`, `server`, and `token` fields.
- All three tools are listed in `registerTools()`.
- `get_eks_kubeconfig` returns `mcp.NewToolResultText`, not `mcp.NewToolResultJSON`.

---

### Phase 6 — Kubernetes Tools with Lazy Client Initialization

#### Why

A Kubernetes client cannot be created at startup because the target cluster is unknown until the agent resolves it at runtime (potentially across multiple EKS clusters in different regions). The lazy `ClientHolder` model solves this: the server starts with no Kubernetes connection and the agent injects a kubeconfig string after calling `get_eks_kubeconfig`. This phase also introduces the `set_kubeconfig` guard pattern — all kube handlers fail fast with a clear message if called before initialization — which prevents silent wrong-cluster operations.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). AWS tooling is complete. Your task is to add Kubernetes pod management tools with a lazy client initialization model — the Kubernetes client is NOT created at startup. Instead, the AI agent injects a kubeconfig string at runtime via a dedicated tool.

<dependencies>
Install before writing code:
  go get k8s.io/client-go
  go get k8s.io/api
  go get k8s.io/apimachinery
</dependencies>

<lazy_init_design>
The Kubernetes client must NOT be initialized at server startup. The intended agent workflow is:

  get_eks_kubeconfig  →  set_kubeconfig  →  any Kubernetes tool

Implement a thread-safe holder:

File: internal/kube/holder.go
  type ClientHolder struct {
      mu     sync.RWMutex
      client Client   // the kube.Client interface
  }
  - Set(kubeconfig string) error
      Parses the kubeconfig string (not a file path) via clientcmd.RESTConfigFromKubeConfig.
      Atomically replaces the held client. On error, preserves the existing client unchanged.
  - Get() (Client, bool)
      Returns the current client and whether it has been initialized.
</lazy_init_design>

<kube_client: internal/kube/>
Define a Client interface (client.go or kube.go) with methods:
  ListPods, GetPod, CreatePod, UpdatePod, DeletePod, GetPodLogs
(exact signatures: refer to k8s.io/api/core/v1 and k8s.io/client-go/kubernetes)

Implement a concrete unexported type. NewClient(kubeconfig *string) returns the interface:
  - nil pointer  → in-cluster config (rest.InClusterConfig)
  - non-nil      → parse kubeconfig CONTENT via clientcmd.RESTConfigFromKubeConfig
</kube_client>

<tools: internal/tools/kube.go>
Implement these tools, all taking *kube.ClientHolder:
1. set_kubeconfig  — required: kubeconfig (string). Calls holder.Set. Returns success/error text.
2. list_pods       — required: namespace. Optional: label_selector.
3. get_pod         — required: namespace, name.
4. create_pod      — required: namespace, manifest (JSON string).
5. update_pod      — required: namespace, name, manifest (JSON string).
6. delete_pod      — required: namespace, name.
7. get_pod_logs    — required: namespace, name. Optional: container, tail_lines (int).

Every handler (except set_kubeconfig) must call holder.Get() at the top and return
mcp.NewToolResultError("kubernetes client not initialized: call set_kubeconfig first")
if the client is not yet set.

RegisterKube(s *server.MCPServer, holder *kube.ClientHolder) wires all seven tools.
</tools>

<main_go_update>
Remove any kubeconfig loading from main.go (no KUBECONFIG_PATH env var).
Pass &kube.ClientHolder{} (zero value, uninitialized) to mcp.NewServer.
Update internal/mcp/server.go: replace KubeClient field with KubeHolder *kube.ClientHolder.
</main_go_update>

<output_types: pkg/types/kube.go>
Define Pod and PodList output structs with clean JSON tags.
</output_types>

<constraints>
- ClientHolder.Set must preserve the existing client if parsing fails.
- All kube tool handlers must guard against uninitialized client.
- internal/tools/kube.go must not import internal/tools/eks.go or any other tools file.
- No kubeconfig file path logic in main.go — the holder starts empty.
</constraints>
```

#### Verification Points

- `go build ./...` succeeds.
- `ClientHolder.Set`: if called with an invalid kubeconfig string, `Get()` still returns the previously set client (atomicity check).
- All kube handlers (except `set_kubeconfig`) return the "not initialized" error message when called before `set_kubeconfig`.
- `main.go` contains no reference to `KUBECONFIG_PATH` or any kubeconfig file loading.
- `internal/mcp/server.go`: the struct field is `KubeHolder *kube.ClientHolder`, not a `kube.Client`.
- `registerTools()` calls `tools.RegisterKube(s, s.kubeHolder)`.

---

### Phase 7 — Kubernetes Tests

#### Why

The `ClientHolder` has two subtleties that are easy to break silently: the atomicity guarantee (a failed `Set` must not clear the existing client) and thread safety under concurrent reads and writes. Unit tests are the only reliable way to verify both without a real cluster. The handler-level tests additionally confirm the "not initialized" guard actually fires — something that cannot be caught by the compiler and would otherwise only surface at runtime against a live agent session.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). The Kubernetes tooling with lazy initialization is in place. Your task is to write tests covering the ClientHolder and the kube tool handlers.

<test_file_1: internal/kube/holder_test.go>
Package: kube_test

Write table-driven or individual tests covering:
1. Get before Set returns (nil, false).
2. Set with an invalid kubeconfig string returns an error; subsequent Get still returns (nil, false).
3. Set with a valid single-cluster kubeconfig succeeds; Get returns (client, true).
4. Switch from cluster A to cluster B (two valid kubeconfigs).
5. Switch back from B to A (reconnect to previous cluster).
6. A failed Set (bad kubeconfig) after a successful Set does NOT clear the existing client.
7. Concurrent reads and a write do not race (use -race flag).

For valid kubeconfig strings, use minimal YAML with a fake HTTPS server URL — no real cluster needed:
  apiVersion: v1
  kind: Config
  clusters:
  - cluster:
      server: https://fake-cluster-a.example.com
      insecure-skip-tls-verify: true
    name: cluster-a
  contexts:
  - context:
      cluster: cluster-a
      user: user-a
    name: ctx-a
  current-context: ctx-a
  users:
  - name: user-a
    user:
      token: fake-token
</test_file_1>

<test_file_2: internal/tools/kube_connection_test.go>
Package: tools  ← MUST be internal (package tools, not package tools_test) so you can call unexported handler functions directly without routing through the MCP server infrastructure.

Write an invoke helper:
  func invoke(h server.ToolHandlerFunc, args map[string]any) (*mcp.CallToolResult, error) {
      req := mcp.CallToolRequest{}
      req.Params.Arguments = args
      return h(context.Background(), req)
  }

Write tests covering:
1. set_kubeconfig with missing argument returns error.
2. set_kubeconfig with invalid kubeconfig returns error.
3. set_kubeconfig with valid cluster A kubeconfig succeeds.
4. After connecting to A, set_kubeconfig with valid cluster B kubeconfig succeeds (switch).
5. Multiple cluster switches in sequence succeed.
6. A failed set_kubeconfig after a successful one preserves the existing client (list_pods still works conceptually — or verify Get still returns true).
7. Calling list_pods before any set_kubeconfig returns the "not initialized" error message.
</test_file_2>

<constraints>
- Do NOT use real AWS or Kubernetes clusters — fake server URLs and tokens are sufficient.
- Do NOT route through s.CallTool() or any MCP server infrastructure — call handler functions directly.
- Run tests with: go test ./internal/kube/... ./internal/tools/... -race
</constraints>
```

#### Verification Points

- `go test ./internal/kube/... -race` passes with 0 failures.
- `go test ./internal/tools/... -race` passes with 0 failures.
- `holder_test.go` uses `package kube_test` (external test package).
- `kube_connection_test.go` uses `package tools` (internal test package — required to access handler funcs).
- The concurrent access test uses goroutines with `sync.WaitGroup` and verifies no data races under `-race`.
- No test makes real network connections.

---

### Phase 8 — Unified Configuration Package

#### Why

The server needs two distinct kinds of configuration: AI credentials (best kept in a YAML file that can be mounted as a Kubernetes Secret) and AWS settings (best read from environment variables, which is the standard AWS convention). A single `pkg/config` package that merges both sources — with env vars taking precedence over YAML — gives operators full flexibility without scattering config logic across multiple files. Centralising this also ensures `main.go` stays thin and that there is exactly one place to add a new config key in future.

#### Prompt

```
You are working in an existing Go project (github.com/muraduiurie/aws-ai-agent, Go 1.25). Your task is to create a unified configuration package that reads a YAML file for AI settings and merges environment variables for AWS settings.

<dependencies>
  go get github.com/spf13/viper
Run go mod tidy after installing.
</dependencies>

<package: pkg/config/config.go>
Implement:

Struct hierarchy:
  Config
    AI  AIConfig    (mapstructure:"ai")
      Claude  ClaudeConfig  (mapstructure:"claude")
        Token    string  (mapstructure:"token")
        MaxToken int     (mapstructure:"maxToken")
        Model    string  (mapstructure:"model")
    AWS AWSConfig   (mapstructure:"aws")
      Profile  string  (mapstructure:"profile")
      Region   string  (mapstructure:"region")
      ReadOnly bool    (mapstructure:"readOnly")
        ↑ Add a detailed comment explaining ReadOnly is loaded and logged at startup
          but is NOT YET ENFORCED — no handler checks it. Before adding any mutating tool,
          enforce this flag in registerTools() or at the handler level.

Load() (*Config, error):
  - Read YAML from the path in CONFIG_PATH env var; default: /etc/aws-ai-agent/config.yaml.
  - Use viper to parse the YAML into Config.
  - After unmarshaling, call mergeAWSEnvVars(cfg.AWS) to override AWS fields from env vars.

mergeAWSEnvVars(base AWSConfig) AWSConfig:
  Precedence: env var > YAML > built-in default.
  - AWS_REGION or AWS_DEFAULT_REGION → Region (AWS_REGION takes priority; default: "eu-west-3").
  - AWS_PROFILE → Profile.
  - READ_ONLY=true (string) → ReadOnly = true.
</package>

<config_example: config.example.yaml>
Create at repo root:
  ai:
    claude:
      token: "your-anthropic-api-key"
      maxToken: 4096
      model: "claude-opus-4-6"

  aws:
    region: "eu-west-3"
    profile: ""
    # readOnly: when true, is intended to block any tool that mutates AWS or
    # Kubernetes resources (e.g. create/update/delete). Currently loaded and
    # logged at startup but not yet enforced — no handler checks it today.
    readOnly: false
</config_example>

<update: cmd/server/main.go>
Replace env-var reading with:
  cfg, err := config.Load()
  if err != nil { log.Fatalf("failed to load config: %v", err) }
Pass &cfg.AWS to aws.NewFactory (update its signature to accept *config.AWSConfig).
Log cfg.AWS.ReadOnly at startup.
Remove the old internal/config import.
</update>

<update: internal/aws/client.go>
Change NewFactory to accept *config.AWSConfig instead of separate region/profile strings.
</update>

<constraints>
- Delete internal/config/ entirely after migrating its logic into pkg/config.
- go mod tidy must leave anthropic-sdk-go and viper as direct (not indirect) dependencies.
- The CONFIG_PATH env var must default to /etc/aws-ai-agent/config.yaml.
- ReadOnly must have a comment at its declaration site explaining it is unenforced.
</constraints>
```

#### Verification Points

- `go build ./...` succeeds with no errors.
- `go mod tidy` leaves `github.com/spf13/viper` as a direct dependency (not `// indirect`).
- `internal/config/` directory no longer exists.
- `pkg/config/config.go`: `AWSConfig.ReadOnly` has a multi-line comment explaining it is unenforced.
- `mergeAWSEnvVars`: env vars take precedence over YAML values (test by setting `AWS_REGION` and verifying it overrides the YAML `region`).
- `config.example.yaml` exists at repo root and matches the YAML structure exactly.
- `main.go` imports `pkg/config`, not `internal/config`.
- Running `CONFIG_PATH=config.example.yaml go run ./cmd/server` loads without error (will block on stdin).

---

## 2. AI Agent

> **Status: Pending** — To be filled after the AI Agent development phase is complete.

### Prompt

*(Will cover: `pkg/agent` package, `Agent` interface, `claudeAgent` implementation, `Ping` connectivity check, wiring into `main.go`.)*

### Verification Points

*(Will cover: Anthropic SDK initialization, `Ping` success/failure, config-driven model/token params.)*

---

## 3. Agentic Automation Loop

> **Status: Pending** — To be filled after the orchestration layer development is complete.

*(This section will document the end-to-end automation loop where the AI agent drives the MCP tools to fulfill complex multi-step tasks: querying EKS, fetching kubeconfig, initializing the Kubernetes client, and performing pod operations — all autonomously in response to a high-level instruction.)*

### Prompt

*(Will cover: agentic loop design, tool-calling strategy, error recovery, state management across tool calls.)*

### Verification Points

*(Will cover: end-to-end scenario tests, idempotency checks, error propagation from tools back to the agent.)*
