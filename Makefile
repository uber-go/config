PACKAGES := $(shell glide novendor)
TEST := go test -race

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
	$(TEST) -cover -coverprofile=coverage.txt $(PACKAGES)
	# bash <(curl -s https://codecov.io/bash)
else
	$(TEST) $(PACKAGES)
endif
