BINARY_NAME ?= docker-machine-driver-makecloud

.PHONY: build
build:
	go build -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

.PHONY: install
install:
	go install ./cmd/$(BINARY_NAME)

.PHONY: test
test:
	go test ./...
