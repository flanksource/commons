.PHONY: test
test:
	go test ./... -v --count=1

.PHONY: lint
lint:
	golangci-lint run