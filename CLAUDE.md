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
