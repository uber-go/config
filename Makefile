PACKAGES := $(shell glide novendor)
TEST := go test -race $(PACKAGES)

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	glide install

.PHONY: test
test:
	go test -race $(PACKAGES) -count 2

.PHONY: ci
ci: SHELL := /bin/bash
ci:
ifdef COVER
	$(TEST) -coverprofile=coverage.txt -covermode=atomic -count 2
	bash <(curl -s https://codecov.io/bash)
else
	$(TEST)
endif
