# Official CLI Parity

Source of truth: <https://crankuptheamps.com/docs/amps-user-guide/utilities/spark>

This matrix tracks the official `spark`-compatible surface that `gofer` intentionally implements.
The goal is semantic, user-visible parity for the six documented `spark` commands.

## Shared Transport Flags

| Flag | Status | Notes |
|---|---|---|
| `-server` | Implemented | Accepts bare `host:port`, `user:pass@host:port`, and full URIs. |
| `-proto` / `-prot` | Implemented | Normalized to AMPS URI/message-type semantics. |
| `-type` | Implemented | Supports spark-style message type selection. |
| `-secure` | Implemented | Promotes the URI scheme to `tcps`. |
| `-urischeme` | Partial | `tcp` and `tcps` are supported; unsupported custom schemes fail fast. |
| `-uriopts` | Partial | `tcp_nodelay`, `tcp_sndbuf`, and `tcp_rcvbuf` are honored by `amps-client-go`. |
| `-authenticator` | Partial | Go-native registry; currently supports `default` and `kerberos`. |

## Command Surface

| Command | Status | Implemented Notes |
|---|---|---|
| `ping` | Implemented | Canonical spark-style URI construction and success output. |
| `publish` | Implemented | Adds `-file`, `-delimiter`, `-rate`, and `-delta`. |
| `sow` | Implemented | Adds `-batchsize`, `-copy`, `-format`, `-orderby`, and `-topn`. |
| `subscribe` | Implemented | Adds `-ack`, `-backlog`, `-max_backlog`, `-copy`, `-delta`, and `-format`. |
| `sow_and_subscribe` | Implemented | Adds SOW/query flags plus `-delta`, `-copy`, and `-format`. |
| `sow_delete` | Implemented | Adds `-file` and delete-all default behavior via `1=1` when no filter is supplied. |

## Format Tokens

`gofer` supports the spark format tags:

- `{bookmark}`
- `{command}`
- `{correlation_id}`
- `{data}`
- `{expiration}`
- `{lease_period}`
- `{length}`
- `{sowkey}`
- `{timestamp}`
- `{topic}`
- `{user_id}`

## Intentional Differences

- Output is spark-compatible in meaning, not byte-for-byte identical.
- JVM-only spark extension points are replaced with Go-native behavior.
- Runtime loading of arbitrary custom URI transports is not supported.
