# Change log

## Release v1.0.28

- Bump github.com/docker/docker from 27.5.1+incompatible to 28.0.0+incompatible
- Fix https://github.com/pedropombeiro/qnapexporter/issues/18

## Release v1.0.26

- Remove version from release asset filename

## Release v1.0.25

- Bump golang.org/x/net from 0.36.0 to 0.38.0
- Extend support for linux/arm with v5/v6/v7 support

## Release v1.0.24

- Bump golang.org/x/net from 0.35.0 to 0.36.0
- Add support for linux/arm (32-bit) architecture

## Release v1.0.23

- Bump gopsutil to v4
- Bump packages
- Check requirements before installing qpkg
- Update Go to 1.23

## Release v1.0.22

- Bump packages
- Update Go to 1.23

## Release v1.0.21

- QPKG package for qnapexporter, built and released via github actions

## Release v1.0.20

- Upgrade packages

## Release v1.0.19

- Add the arm64 architecture build step in release workflows

## Release v1.0.18

- Update go and dependencies
- Report metric fetch error with name

## Release v1.0.17

- Upgrade modules
- Lint code

## Release v1.0.16

- Show version and revision in status page

## Release v1.0.15

- Bump github.com/docker/distribution from 2.8.1+incompatible to 2.8.2+incompatible
- Fix: allocationToken out of range
- Bump github.com/docker/docker from 23.0.3+incompatible to 24.0.9+incompatible
- Bump golang.org/x/net from 0.9.0 to 0.23.0
- Upgrade to Go 1.21

## Release v1.0.14

- Add .projections.json file
- Migrate deprecated packages
- Move to GitHub
- Upgrade Go modules

## Release v1.0.13

- Retry UPS call if pipe broken

## Release v1.0.12

- Add error message to healthchecks call

## Release v1.0.11

- Try to fix spikes by forcing read if freeSizeBytes is 0

## Release v1.0.10

- Try to fix spikes in disk used space

## Release v1.0.9

- Fix published package version

## Release v1.0.8

- Fix published package version

## Release v1.0.7

- Upgrade Go modules
- Fix lint errors

## Release v1.0.6

- Add GitHub Actions support

## Release v1.0.5

- Add GitHub Actions support

## Release v1.0.4

- Remove timestamps, to avoid "Error on ingesting samples with different value but same timestamp"

## Release v1.0.3

- Upgrade to Go 1.19.5
- Add read/write cache hits information for QTS 5.0
- Add dm-cache support for QTS 5
- Upgrade to go 1.17
- Ignore when flashcache is not available
- Default to no ping target

## Release v1.0.2

- Ignore when flashcache is not present (e.g. on QTS 5)

## Release v1.0.1

- Version 1.0.0
- Fix CPU hogging when Docker daemon is unreachable
- Use context.Done() instead of os.Signal
- Sleep for 10 seconds after receiving error from Docker client

## Release v0.0.8

- Fix hang at startup and add more tracing
- Discard docker attributes from Docker events
- Expose docker status in status page
- Add support for enclosure fans
- Update UPS dashboard
- Use gopsutil for disk stats
- Fix volume enumeration to ignore single disk volumes
- Create TagExtractor interface
- Add support for posting Docker event Grafana annotations
- Replace github.com/mackerelio/go-osstat with github.com/shirou/gopsutil/v3

## Release v0.0.7

- Update packages
- Use CPU as a counter
- Retry connecting to UPS after 1 hour
- Try to reconnect to UPS in the event of a broken pipe
- Send any error during metric collection to healthcheck service
- Add support for HEALTHCHECK_CONFIG env variable

## Release v0.0.6

- Remove speedtest metrics

## Release v0.0.5

- Fix ULSpeed

## Release v0.0.4

- Show version in usage screen
- Add hourly speed tests
- Add stable timestamps to volume metrics
- Add VS Code debug configuration

## Release v0.0.3

- Add `go_program` metric
- Simplify installation instructions in README.md

## Release v0.0.1

- Initial release
