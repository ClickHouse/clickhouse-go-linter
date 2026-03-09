.PHONY: all
all: fmt vet test build

.PHONY: fmt
fmt: ## run go fmt
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: build
build:
	go build -o bin/clickhouse-go-linter .