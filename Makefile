
build:
	go build
	@cp subtext testing/

install:
	go install

test:
	@go test ./...

.PHONY: build test