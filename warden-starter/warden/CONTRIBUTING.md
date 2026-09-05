# Contributing to Warden

This project is early-stage — see ROADMAP.md for the current milestone.
Right now the most useful contributions are:

1. **M1 groundwork**: a working `internal/policy` package (parse + validate
   the YAML schema in ARCHITECTURE.md) and an `internal/sandbox/linux`
   package that shells out to `bwrap` with the right bind-mount arguments.
2. **Test servers**: point us at (or write) small, safe MCP servers we can
   use as integration test fixtures for sandboxing behavior.
3. **Schema review**: poke holes in the policy.yaml schema in
   ARCHITECTURE.md before too much code depends on it.

## Development setup

```bash
git clone <this repo>
cd warden
go build ./...
```

(Once M1 lands, this section will grow to include how to run the test suite
against a real bwrap sandbox.)

## Pull requests

Keep PRs scoped to one roadmap item where possible — easier to review, and
easier to roll back if a design assumption turns out wrong.
