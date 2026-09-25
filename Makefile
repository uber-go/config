export GOBIN ?= $(shell pwd)/bin

STATICCHECK = $(GOBIN)/staticcheck

GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

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
lint: gofmt govet staticcheck

.PHONY: govet
govet:
	@echo "Running govet"
	@go vet ./...

.PHONY: gofmt
gofmt:
	@echo "Running gofmt"
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@gofmt -e -s -l $(GO_FILES) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || (echo "gofmt failed:" | cat - $(FMT_LOG) && false)

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	@echo "Running staticcheck"
	@$(STATICCHECK) ./...

$(STATICCHECK):
	@echo "Installing staticcheck"
	@cd tools && go install honnef.co/go/tools/cmd/staticcheck
