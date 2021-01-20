.PHONY: build
build:
	@ mkdir -p ./bin
	@ go build -o bin/qnapexporter .

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
