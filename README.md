# gofer — AMPS command-line client (Go edition)

<p align="left">
  <a href="https://github.com/Thejuampi/gofer/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/Thejuampi/gofer/ci.yml?branch=main&label=CI&logo=githubactions&logoColor=white"></a>
  <a href="https://github.com/Thejuampi/gofer/actions/workflows/release.yml"><img alt="Release" src="https://img.shields.io/github/actions/workflow/status/Thejuampi/gofer/release.yml?label=release&logo=github"></a>
  <a href="https://github.com/Thejuampi/gofer/releases"><img alt="Latest Release" src="https://img.shields.io/github/v/release/Thejuampi/gofer?sort=semver&logo=github"></a>
  <a href="https://github.com/Thejuampi/gofer/blob/main/go.mod"><img alt="Go Version" src="https://img.shields.io/github/go-mod/go-version/Thejuampi/gofer?logo=go"></a>
  <a href="https://github.com/Thejuampi/gofer/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/Thejuampi/gofer"></a>
</p>

**gofer** is a cross-platform CLI for interacting with [AMPS](https://www.cranktheamps.com/) instances.
It compiles to a single native binary with zero external dependencies, built on top of
[amps-client-go](https://github.com/Thejuampi/amps-client-go) — the high-performance Go AMPS client.

---

## Installation

```bash
go install github.com/Thejuampi/gofer@latest
```

Or download a pre-built binary from [Releases](https://github.com/Thejuampi/gofer/releases).

---

## Commands

| Command | Description |
|---|---|
| `ping` | Test connectivity to an AMPS instance |
| `publish` | Publish a message to a topic |
| `subscribe` | Subscribe to a topic and stream messages |
| `sow` | Query the State-of-the-World for a topic |
| `sow_and_subscribe` | SOW snapshot followed by live subscription |
| `sow_delete` | Delete records from a SOW topic |

Run `gofer <command> -help` for flag details.

---

## Quick Start

```bash
# Check connectivity
gofer ping -server tcp://localhost:9007/amps/json

# Publish
gofer publish -server tcp://localhost:9007/amps/json -topic orders -data '{"id":1}'

# Subscribe (stream until Ctrl-C)
gofer subscribe -server tcp://localhost:9007/amps/json -topic orders

# Subscribe, receive exactly 5 messages and exit
gofer subscribe -server tcp://localhost:9007/amps/json -topic orders -n 5

# SOW query
gofer sow -server tcp://localhost:9007/amps/json -topic orders -filter '/id > 10'

# SOW + live subscription (stop after 20 messages)
gofer sow_and_subscribe -server tcp://localhost:9007/amps/json -topic orders -n 20

# Delete from SOW
gofer sow_delete -server tcp://localhost:9007/amps/json -topic orders -filter '/id = 1'
```

---

## Building from Source

Requires a clone of [amps-client-go](https://github.com/Thejuampi/amps-client-go) as a sibling directory
(the `go.mod` `replace` directive points to `../amps-client-go`).

```bash
# layout
repos/
  amps-client-go/   # https://github.com/Thejuampi/amps-client-go
  gofer/            # this repo

cd gofer
make build          # produces ./gofer (or gofer.exe on Windows)
make test           # integration tests (requires fakeamps from amps-client-go)
```

---

## Related

- **[amps-client-go](https://github.com/Thejuampi/amps-client-go)** — the Go AMPS client library powering this tool
- **[AMPS](https://www.cranktheamps.com/)** — 60east's ultra-low-latency message broker

---

## License

[MIT](LICENSE)
