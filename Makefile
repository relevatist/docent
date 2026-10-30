BIN := bin/docent

.PHONY: all
all: check build

.PHONY: build
build: plugin
	go build -o $(BIN) ./cmd/docent

.PHONY: install
install: plugin
	go install ./cmd/docent

# Includes the integration test, which runs when zellij and hx are on PATH.
.PHONY: test
test: plugin
	go test -count=1 ./...

.PHONY: test-unit
test-unit: plugin
	go test -count=1 -short ./...

.PHONY: vet
vet: plugin
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

.PHONY: lint
lint: plugin
	golangci-lint run

# @latest, so govulncheck is built with the current Go toolchain.
.PHONY: vuln
vuln: plugin
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: check
check: plugin fmt-check vet lint test

# go:embed needs assets/zellij/docent.wasm, so the Go targets build it first.
.PHONY: plugin
plugin:
	cd plugin && cargo build --locked --release --target wasm32-wasip1
	install -m 644 plugin/target/wasm32-wasip1/release/docent.wasm assets/zellij/docent.wasm

.PHONY: clean
clean:
	rm -rf bin
