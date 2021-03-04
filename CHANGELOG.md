# Release v0.0.8

- Fix hang at startup if a volume was ignored
- Add more tracing when reading environment

# Release v0.0.7

- Fix CPU metrics
- Add support for HEALTHCHECK_CONFIG env variable
- Send any error during metric collection to healthcheck service
- Try to reconnect to UPS in the event of a broken pipe
- Retry connecting to UPS after 1 hour

# Release v0.0.6

Remove speedtest metrics as it was causing lost metrics (probably due to timeout).

# Release v0.0.5

Fix speedtest metrics reported as 0.

# Release v0.0.4

Add speedtest metrics.

# Release v0.0.3

Add `go_program` metric.

# Release v0.0.1

This is the first public release of the QNAP exporter.
