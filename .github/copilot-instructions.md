# Copilot Instructions — gofer

## Big Picture

**gofer** is a standalone CLI tool for interacting with AMPS instances.
It is built on top of [amps-client-go](https://github.com/Thejuampi/amps-client-go) — the Go AMPS client library.

- Module: `github.com/Thejuampi/gofer`
- Single `package main` at repo root
- Depends on: `github.com/Thejuampi/amps-client-go` (replaced locally via `../amps-client-go` in development)

## File Map

| File | Role |
|---|---|
| `main.go` | Entry point, command dispatch, usage string |
| `connect.go` | `connect(uri, timeout)` helper — creates, connects, and logons a client |
| `ping.go` | `ping` subcommand |
| `publish.go` | `publish` subcommand |
| `subscribe.go` | `subscribe` subcommand |
| `sow.go` | `sow` subcommand |
| `sow_and_subscribe.go` | `sow_and_subscribe` subcommand |
| `sow_delete.go` | `sow_delete` subcommand |
| `output.go` | Buffered stdout writer, `writeMessage`, `flushOutput` |
| `gofer_test.go` | Integration tests — builds fakeamps + gofer binaries and runs end-to-end |

## Developer Workflow

```bash
make build    # go build -o gofer .
make test     # go test -count=1 -timeout 120s .
make fmt      # go fmt ./...
make vet      # go vet ./...
make tidy     # go mod tidy
make release  # vet + test + build
```

Requires `../amps-client-go` to be present (sibling directory) for:
- The `replace` directive in `go.mod`
- Building `fakeamps` during integration tests

## Coding Conventions

- Single `package main` — no sub-packages needed for a CLI this size.
- Each subcommand lives in its own file (`ping.go`, `subscribe.go`, etc.).
- `connect.go` is the only place that creates an `amps.Client` — all subcommands call `connect(server, timeout)`.
- Use `var` for local variable declarations.
- Error messages use `fmt.Errorf` with the subcommand name as prefix: `fmt.Errorf("subscribe: %w", err)`.
- Output always goes through the buffered `writer` in `output.go`; call `flushOutput()` in `defer`.
- KISS — one file per command, no abstraction layers.

## Testing Conventions

- `gofer_test.go` is an **integration test** — it compiles both `fakeamps` (from `../amps-client-go/tools/fakeamps`) and `gofer` (from `.`) into temp binaries, starts a real fakeamps server, and exercises the CLI end-to-end.
- `repoRoot(t)` finds this module's root.
- `ampsClientGoRoot(t)` finds `../amps-client-go` — used to build fakeamps.
- Keep tests as black-box CLI tests: assert stdout/stderr/exit code only.
