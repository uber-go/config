PACKAGES := $(shell glide novendor)

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	glide install

.PHONY: test
test:
	go test -race $(PACKAGES)

.PHONY: ci
ci: SHELL := /bin/bash
ci:
	go test -race $(PACKAGES) -coverprofile=coverage.txt -covermode=atomic
	bash <(curl -s https://codecov.io/bash)

.PHONY: license
license:
	$(ECHO_V)./.build/license.sh
