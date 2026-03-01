HX_DIR := cmd/hx

.PHONY: build
build:
	$(MAKE) -C $(HX_DIR) build

.PHONY: install
install:
	$(MAKE) -C $(HX_DIR) install

.PHONY: test
test:
	go test ./... -v --count=1

.PHONY: lint
lint:
	mkdir -p bin
	GOBIN=$(shell realpath bin) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1
	./bin/golangci-lint run

.PHONY: tidy
tidy:
	go mod tidy

