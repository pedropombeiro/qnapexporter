PKG = gitlab.com/pedropombeiro/qnapexporter
VERSION_PKG = $(PKG)/lib/exporter/prometheus
PACKAGE_VERSION ?= 'dev'
REVISION := $(shell git rev-parse --short=8 HEAD || echo unknown)
BRANCH := $(shell git show-ref | grep "$(REVISION)" | grep -v HEAD | awk '{print $$2}' | sed 's|refs/remotes/origin/||' | sed 's|refs/heads/||' | sort | head -n 1)
BUILT := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)

GO_LDFLAGS ?= -X $(VERSION_PKG).REVISION=$(REVISION) -X $(VERSION_PKG).BUILT=$(BUILT) \
              -X $(VERSION_PKG).BRANCH=$(BRANCH) -X $(VERSION_PKG).VERSION=$(PACKAGE_VERSION) \
              -s -w

.PHONY: build
build:
	@ mkdir -p ./bin
	go build -ldflags "$(GO_LDFLAGS)" -o bin/qnapexporter .

.PHONY: test
test:
	@ go test ./...

.PHONY: mocks
mocks:
	@ find . -name mock_*.go -delete
	@ mockery --dir=. --recursive --all --inpackage

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
