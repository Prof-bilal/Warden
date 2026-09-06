# Contributing to Warden

This project is early-stage — see ROADMAP.md for the current milestone.
Right now the most useful contributions are:

1. **Platform backends**: M4 Seatbelt/Docker work on macOS and M5
   AppContainer, Windows Filtering Platform, ETW, and Job Object work on
   Windows. Every backend must fail closed when an enforcement primitive is
   unavailable.
2. **Test servers**: point us at (or write) small, safe MCP servers we can
   use as integration test fixtures for sandboxing behavior.
3. **Schema review**: poke holes in the policy YAML schema in
   ARCHITECTURE.md before too much code depends on it.

## Development setup

```bash
git clone <this repo>
cd warden
go build ./...
go vet ./...
go test ./...
```

Before changing shared code, also verify that it cross-builds:

```bash
GOOS=darwin GOARCH=arm64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```

## Pull requests

Keep PRs scoped to one roadmap item where possible — easier to review, and
easier to roll back if a design assumption turns out wrong.
