.PHONY: test
test:
	go test ./... -v --count=1

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy	

