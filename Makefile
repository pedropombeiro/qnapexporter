test:
	@ go test ./...

mocks:
	@ find . -name mock_*.go -delete
	@ mockery --dir=. --recursive --all --inpackage
