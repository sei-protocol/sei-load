# Loadtest_v2 Makefile
# Generates Go bindings for smart contracts and builds the seiload CLI

# Directories
CONTRACTS_DIR := generator/contracts
SCENARIOS_DIR := generator/scenarios
BINDINGS_DIR := generator/bindings
BUILD_DIR := build

# Binary configuration
BINARY_NAME := seiload
INSTALL_PATH := $(GOPATH)/bin
ifeq ($(GOPATH),)
	INSTALL_PATH := $(HOME)/go/bin
endif

# Tools
SOLC := /tmp/solc
ABIGEN := abigen
NVM_DIR := $(HOME)/.nvm
NODE_VERSION := 20

# Pinned solc release + integrity hash (supply-chain). Single source of truth
# for both `setup-node` and the CI download step. Verified against the official
# Solidity release index https://binaries.soliditylang.org/linux-amd64/list.json
# (0.8.19 -> sha256 0x7a5c1d3d...cd9eb48), which matches the GitHub
# solc-static-linux artifact. Bump version + hash together.
SOLC_VERSION := 0.8.19
SOLC_SHA256 := 7a5c1d3dc9a8eba62bb2ec37192c9178ae5fe8a54a56e5573fd3c9c17cd9eb48

# EVM target for solc. `paris` is solc 0.8.19's highest supported target (its
# implicit default), so we pin it explicitly to make that default a written
# invariant: a future solc bump can't silently emit newer opcodes (e.g.
# PUSH0/MCOPY/TSTORE) and shift the bytecode/gas surface under us. paris is a
# strict subset of Sei's Cancun/Pectra-era forks (paris ⊂ Sei), so paris-targeted
# bytecode is unconditionally safe to deploy; runtime gas is set by the chain's
# active fork regardless of compile target, so the target never distorts
# measurements.
SOLC_EVM_VERSION := paris

# go-ethereum version sourced from go.mod (pins abigen for reproducible bindings).
# Falls back to grepping go.mod if `go list` is unavailable.
GETH_VERSION := $(shell go list -m -f '{{.Version}}' github.com/ethereum/go-ethereum 2>/dev/null || grep -E 'github.com/ethereum/go-ethereum ' go.mod | awk '{print $$2}')

# Pinned golangci-lint version. Keep in sync with the workflow `version:` and
# `.golangci.yml` (bump all three together); an unpinned `latest` drifts into a
# "passes locally, fails CI" trap. See README "Before you push".
GOLANGCI_VERSION := 2.12.2

# Find all .sol files in contracts directory
SOL_FILES := $(wildcard $(CONTRACTS_DIR)/*.sol)
CONTRACT_NAMES := $(basename $(notdir $(SOL_FILES)))

# Generated files
ABI_FILES := $(addprefix $(BUILD_DIR)/, $(addsuffix .abi, $(CONTRACT_NAMES)))
BIN_FILES := $(addprefix $(BUILD_DIR)/, $(addsuffix .bin, $(CONTRACT_NAMES)))
BINDING_FILES := $(addprefix $(BINDINGS_DIR)/, $(addsuffix .go, $(CONTRACT_NAMES)))
SCENARIO_TEMPLATE_FILES := $(addprefix $(SCENARIOS_DIR)/, $(addsuffix .go, $(CONTRACT_NAMES)))

.PHONY: generate generate-bindings check-bindings install-abigen install-lint clean help build-cli install setup-node build test lint verify

# Default target
help:
	@echo "Available targets:"
	@echo "  verify            - Run the gating CI checks: lint + test + build + CLI --help + check-bindings"
	@echo "  build             - Build the seiload CLI (alias for build-cli)"
	@echo "  test              - Run tests with coverage (race detector enabled)"
	@echo "  lint              - Run linting and static analysis (golangci-lint $(GOLANGCI_VERSION))"
	@echo "  setup-node        - Install nvm, Node.js 20, and solc"
	@echo "  generate          - Generate Go bindings and scenario templates for all contracts"
	@echo "  generate-bindings - Regenerate ONLY the Go bindings (no scenarios/factory)"
	@echo "  check-bindings    - Fail if committed bindings are out of sync with contracts"
	@echo "  install-tools     - Install the full pinned toolchain (solc, abigen, golangci-lint)"
	@echo "  install-abigen    - Install abigen pinned to the go.mod go-ethereum version"
	@echo "  install-lint      - Install golangci-lint pinned to $(GOLANGCI_VERSION)"
	@echo "  clean             - Remove generated files"
	@echo "  help              - Show this help message"
	@echo "  build-cli         - Build the seiload CLI"
	@echo "  install           - Install the seiload CLI"
	@echo ""
	@echo "Before pushing: run 'make verify' (local CI parity). Run 'make install-tools'"
	@echo "first to get the pinned toolchain (golangci-lint $(GOLANGCI_VERSION) etc.)."

# Setup Node.js environment with nvm
setup-node:
	@echo "🔧 Setting up Node.js environment..."
	@if [ ! -d "$(NVM_DIR)" ]; then \
		echo "📦 Installing nvm..."; \
		curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.4/install.sh | bash; \
		echo "🔄 Sourcing nvm for current session..."; \
		export NVM_DIR="$(HOME)/.nvm"; \
		[ -s "$$NVM_DIR/nvm.sh" ] && . "$$NVM_DIR/nvm.sh"; \
	else \
		echo "✅ nvm already installed"; \
	fi
	@echo "🔧 Setting up Node.js $(NODE_VERSION)..."
	@export NVM_DIR="$(HOME)/.nvm" && \
	[ -s "$$NVM_DIR/nvm.sh" ] && . "$$NVM_DIR/nvm.sh" && \
	nvm install $(NODE_VERSION) && \
	nvm use $(NODE_VERSION)
	@echo "📦 Installing native solc binary..."
	@curl --fail --proto '=https' --tlsv1.2 -L https://github.com/ethereum/solidity/releases/download/v$(SOLC_VERSION)/solc-static-linux -o /tmp/solc
	@echo "🔒 Verifying solc sha256..."
	@echo "$(SOLC_SHA256)  /tmp/solc" | sha256sum -c -
	@chmod +x /tmp/solc
	@echo "✅ Node.js environment setup complete"
	@echo "ℹ️  Note: You may need to restart your shell or run 'source ~/.bashrc' to use nvm in new sessions"

# Main generate target
generate: $(BINDING_FILES) $(SCENARIO_TEMPLATE_FILES)
	@echo "🏭 Updating scenario factory..."
	@./scripts/update_factory.sh $(CONTRACT_NAMES)
	@echo "✅ Generated bindings and scenario templates for contracts: $(CONTRACT_NAMES)"

# Bindings-only target: rebuilds the .sol -> .abi/.bin -> binding chain WITHOUT
# touching human-edited scenario templates or the scenario factory. This is the
# "write a contract, regenerate its binding" path that CI validates.
generate-bindings: $(BINDING_FILES)
	@echo "✅ Generated bindings for contracts: $(CONTRACT_NAMES)"

# Create build directory
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Create bindings directory
$(BINDINGS_DIR):
	@mkdir -p $(BINDINGS_DIR)

# Create scenarios directory
$(SCENARIOS_DIR):
	@mkdir -p $(SCENARIOS_DIR)

# Compile a single contract to ABI and bytecode.
# --evm-version pins the target (see SOLC_EVM_VERSION above).
# --metadata-hash none strips the trailing CBOR metadata hash so bytecode is
# reproducible across repo paths / build hosts (the hash embeds source paths)
# and is slightly smaller; it does not affect the ABI or function selectors.
$(BUILD_DIR)/%.abi $(BUILD_DIR)/%.bin: $(CONTRACTS_DIR)/%.sol | $(BUILD_DIR)
	@echo "🔨 Compiling contract: $*"
	@$(SOLC) --abi --bin --optimize --evm-version $(SOLC_EVM_VERSION) --metadata-hash none --overwrite -o $(BUILD_DIR) $<
	@echo "✅ Compiled: $*"

# Generate Go binding from ABI and bytecode
$(BINDINGS_DIR)/%.go: $(BUILD_DIR)/%.abi $(BUILD_DIR)/%.bin | $(BINDINGS_DIR)
	@echo "🏭 Generating Go binding: $*"
	@$(ABIGEN) --abi=$(BUILD_DIR)/$*.abi --bin=$(BUILD_DIR)/$*.bin --pkg=bindings --type=$* --out=$@
	@echo "✅ Generated binding: $*"

# Generate scenario template files (only if they don't exist)
$(SCENARIOS_DIR)/%.go: | $(SCENARIOS_DIR)
	@./scripts/generate_scenario_template.sh $* $@

# Clean generated files
clean:
	@echo "🧹 Cleaning generated files ..."
	@rm -rf $(BUILD_DIR) $(BINDINGS_DIR)
	@echo "✅ Cleaned up generated files"

# Check if required tools are installed
check-tools:
	@echo "🔍 Checking required tools ..."
	@which $(SOLC) > /dev/null || (echo "❌ solc not found. Run 'make setup-node' to install" && exit 1)
	@which $(ABIGEN) > /dev/null || (echo "❌ abigen not found. Run 'make install-tools' to install" && exit 1)
	@echo "✅ All required tools are available"

# Install abigen pinned to the go.mod go-ethereum version.
# Pinning (not @latest) keeps binding output reproducible so CI drift checks
# don't flake when go-ethereum publishes a new release.
install-abigen:
	@echo "📦 Installing abigen@$(GETH_VERSION) ..."
	@test -n "$(GETH_VERSION)" || (echo "❌ could not resolve go-ethereum version from go.mod" && exit 1)
	@go install github.com/ethereum/go-ethereum/cmd/abigen@$(GETH_VERSION)
	@echo "✅ Installed abigen@$(GETH_VERSION)"

# Drift check: regenerate bindings from source and fail if they differ from
# what is committed. We force a clean rebuild (-B) so Make's mtime logic cannot
# skip regeneration — in CI a freshly-checked-out committed binding can be newer
# than the rebuilt .abi/.bin, which would otherwise let stale/tampered output
# pass the gate. `git add -N` stages the *intent to add* any brand-new (untracked)
# binding so `git diff --exit-code` also fails on a never-committed contract.
# This target is developer-facing (see `make help`), so it must be tree-neutral:
# capture the diff result, ALWAYS reset the index for $(BINDINGS_DIR), then fail.
# This leaves `git status` exactly as it was found (no staged add-intent junk).
check-bindings:
	@$(MAKE) -B generate-bindings
	@git add -N -- $(BINDINGS_DIR)
	@git diff --exit-code -- $(BINDINGS_DIR) > /dev/null 2>&1; rc=$$?; \
		git reset -q -- $(BINDINGS_DIR) >/dev/null 2>&1 || true; \
		if [ $$rc -ne 0 ]; then \
			echo ""; \
			echo "❌ Bindings are out of sync with contracts."; \
			echo "   Contracts changed (or a new contract was added) but bindings"; \
			echo "   were not regenerated/committed."; \
			echo "   Run 'make generate-bindings' and commit the result."; \
			echo ""; \
			git --no-pager diff -- $(BINDINGS_DIR); \
			exit 1; \
		fi
	@echo "✅ Bindings are in sync with contracts"

# Install golangci-lint pinned to GOLANGCI_VERSION for CI parity (CI pins the
# same version via golangci-lint-action).
install-lint:
	@echo "📦 Installing golangci-lint@v$(GOLANGCI_VERSION) ..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_VERSION)
	@echo "✅ Installed golangci-lint@v$(GOLANGCI_VERSION)"

# Install tools (optional convenience target)
install-tools: setup-node install-abigen install-lint
	@echo "✅ Tools installation complete"

# Build the seiload CLI binary
build-cli: | $(BUILD_DIR)
	@echo "🔨 Building CLI"
	@go mod tidy
	@go mod download
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Built CLI: $(BUILD_DIR)/$(BINARY_NAME)"

# Install the seiload CLI
install: build-cli
	@echo "📦 Installing CLI ..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Installed CLI: $(BINARY_NAME)"

# Build the seiload CLI binary (alias for build-cli)
build: build-cli

# Run tests with coverage
test:
	@echo "🔍 Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@echo "✅ Tests passed"

# Run linting. Expects golangci-lint == GOLANGCI_VERSION (`make install-lint`);
# warns (not fails) on a mismatch, since a different binary can shift results.
lint:
	@echo "🔍 Running linting and static analysis..."
	@have=$$(golangci-lint version --short 2>/dev/null || golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
		have=$${have#v}; \
		if [ -n "$$have" ] && [ "$$have" != "$(GOLANGCI_VERSION)" ]; then \
			echo "⚠️  golangci-lint $$have on PATH != pinned $(GOLANGCI_VERSION). Run 'make install-lint' for CI parity."; \
		fi
	@golangci-lint run
	@echo "✅ Linting and static analysis passed"

# Local CI parity for the gating jobs (build-and-test.yml + bindings-check.yml).
# Mirrors build-and-test's lint/test/build/--help so a broken main/CLI is caught
# locally, not in CI. The one CI step NOT folded in is the dry-run smoke: it's a
# backgrounded run killed after 5s and never asserts an exit code, so it's a weak,
# non-deterministic signal that's not worth a 5s+ wall-time tax on every verify.
# That step stays CI-only; see README "Before you push".
#
# The gates are invoked as ordered sub-makes (not prerequisites) so `verify`
# runs them sequentially regardless of `-j`. As parallel prerequisites under
# `make -j`, `build` and `check-bindings` would run concurrently and both write
# the shared $(BUILD_DIR) tree (CLI binary vs. contract .abi/.bin from check-
# bindings' `-B generate-bindings`), racing each other. Sub-makes scope the
# serialization to `verify` alone, leaving the rest of the Makefile parallel-
# safe (unlike a global `.NOTPARALLEL`). `&&` short-circuits on first failure
# so a broken gate stops the chain with that gate's non-zero exit.
verify:
	@$(MAKE) lint && $(MAKE) test && $(MAKE) build && $(MAKE) check-bindings
	@echo "🔍 Smoke-testing CLI entrypoint (--help)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --help > /dev/null
	@echo "✅ verify passed (lint + test + build + --help + check-bindings)"
