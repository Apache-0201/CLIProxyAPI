# fork/v7.1.5

## Changes

- capture first-token latency at the HTTP layer and propagate it through `usage.Record`
- persist `first_token_latency_ms` in both SQLite and PostgreSQL usage storage
- expose first-token latency in monitor request log queries and memory snapshots
- change monitor request log `TotalDurationMs` to display first-token latency
- update per-request `tok/s` calculation to use `output / (total - ttft)` by default, but fall back to `output / total` when TTFT is missing or dominates total latency (>= 80%)
- add regression tests for first-chunk capture, usage propagation, monitor display, and `tok/s` calculation

## Verification

- `gofmt -w` on all modified Go files
- `go build -o /tmp/cli-proxy-api-verify ./cmd/server && rm /tmp/cli-proxy-api-verify`
- `go test ./internal/api/... ./internal/usage/...`
