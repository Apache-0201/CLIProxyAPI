# Changelog

## fork/v7.1.6 - 2026-04-19

### Changed

- Updated monitor request-log `tok/s` calculation to use output tokens divided by total latency minus first-token latency.
- Kept total latency as the denominator when the gap between total latency and first-token latency is large, preventing streaming requests from overstating throughput.
- Covered the new throughput behavior with management monitor unit tests.
