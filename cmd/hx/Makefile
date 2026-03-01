.PHONY: build install test fixtures

VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o .bin/hx .

install:
	go install .

test:
	go test ./... -v -count=1

fixtures:
	go run github.com/flanksource/gavel/cmd/gavel@latest fixtures fixtures/hx.md
