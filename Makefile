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

# go-ethereum version sourced from go.mod (pins abigen for reproducible bindings).
# Falls back to grepping go.mod if `go list` is unavailable.
GETH_VERSION := $(shell go list -m -f '{{.Version}}' github.com/ethereum/go-ethereum 2>/dev/null || grep -E 'github.com/ethereum/go-ethereum ' go.mod | awk '{print $$2}')

# Find all .sol files in contracts directory
SOL_FILES := $(wildcard $(CONTRACTS_DIR)/*.sol)
CONTRACT_NAMES := $(basename $(notdir $(SOL_FILES)))

# Generated files
ABI_FILES := $(addprefix $(BUILD_DIR)/, $(addsuffix .abi, $(CONTRACT_NAMES)))
BIN_FILES := $(addprefix $(BUILD_DIR)/, $(addsuffix .bin, $(CONTRACT_NAMES)))
BINDING_FILES := $(addprefix $(BINDINGS_DIR)/, $(addsuffix .go, $(CONTRACT_NAMES)))
SCENARIO_TEMPLATE_FILES := $(addprefix $(SCENARIOS_DIR)/, $(addsuffix .go, $(CONTRACT_NAMES)))

.PHONY: generate generate-bindings check-bindings install-abigen clean help build-cli install setup-node build test lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build             - Build the seiload CLI (alias for build-cli)"
	@echo "  test              - Run tests with coverage"
	@echo "  lint              - Run linting and static analysis"
	@echo "  setup-node        - Install nvm, Node.js 20, and solc"
	@echo "  generate          - Generate Go bindings and scenario templates for all contracts"
	@echo "  generate-bindings - Regenerate ONLY the Go bindings (no scenarios/factory)"
	@echo "  check-bindings    - Fail if committed bindings are out of sync with contracts"
	@echo "  install-abigen    - Install abigen pinned to the go.mod go-ethereum version"
	@echo "  clean             - Remove generated files"
	@echo "  help              - Show this help message"
	@echo "  build-cli         - Build the seiload CLI"
	@echo "  install           - Install the seiload CLI"

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

# Compile a single contract to ABI and bytecode
$(BUILD_DIR)/%.abi $(BUILD_DIR)/%.bin: $(CONTRACTS_DIR)/%.sol | $(BUILD_DIR)
	@echo "🔨 Compiling contract: $*"
	@$(SOLC) --abi --bin --optimize --overwrite -o $(BUILD_DIR) $<
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

# Install tools (optional convenience target)
install-tools: setup-node install-abigen
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

# Run linting and static analysis
lint:
	@echo "🔍 Running linting and static analysis..."
	@golangci-lint run
	@echo "✅ Linting and static analysis passed"
