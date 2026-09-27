export GOBIN ?= $(shell pwd)/bin

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -coverprofile=cover.out -coverpkg=./... -v ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: lint
lint: golangci-lint tidy-lint

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: tidy-lint
tidy-lint:
	go mod tidy
	git diff --exit-code -- go.mod go.sum
