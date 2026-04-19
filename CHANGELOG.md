# Changelog

## fork/v7.1.7 - 2026-04-20

### Changed

- Added durable `auth_identity` bindings for account-bind routing so Codex OAuth credentials stay bound after quota refresh or auth file rewrites change `auth_index`.
- Exposed `auth_identity` from management auth-file responses and resolved identity bindings to the current runtime `auth_index`.
- Kept legacy `auth_index` bindings compatible while adding regression coverage for identity parsing and runtime binding resolution.

## fork/v7.1.6 - 2026-04-19

### Changed

- Updated monitor request-log `tok/s` calculation to use output tokens divided by total latency minus first-token latency.
- Kept total latency as the denominator when the gap between total latency and first-token latency is large, preventing streaming requests from overstating throughput.
- Covered the new throughput behavior with management monitor unit tests.
