
build:
	go build
	@cp subtext testing/

test:
	@go test ./...

.PHONY: build test