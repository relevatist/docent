BIN := bin/docent

.PHONY: all
all: check build

.PHONY: build
build:
	go build -o $(BIN) ./cmd/docent

.PHONY: install
install:
	go install ./cmd/docent

# Includes the integration test, which runs when zellij and hx are on PATH.
.PHONY: test
test:
	go test -count=1 ./...

.PHONY: test-unit
test-unit:
	go test -count=1 -short ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

.PHONY: lint
lint:
	golangci-lint run

# @latest, so govulncheck is built with the current Go toolchain.
.PHONY: vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: check
check: fmt-check vet lint test

.PHONY: clean
clean:
	rm -rf bin
