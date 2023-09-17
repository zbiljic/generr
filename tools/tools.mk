GOPATH_DIR := $(shell go env GOPATH)
TOOLS_BIN_DIR := $(if $(GOPATH_DIR),$(GOPATH_DIR)/bin,$(error ERROR: GOPATH missing))

include tools/versions.mk

GOFUMPT := $(TOOLS_BIN_DIR)/gofumpt
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint

$(TOOLS_BIN_DIR):
	@mkdir -p $(TOOLS_BIN_DIR)

$(GOFUMPT): $(TOOLS_BIN_DIR)
	@GOBIN=$(TOOLS_BIN_DIR) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

$(GOLANGCI_LINT): $(TOOLS_BIN_DIR)
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || ( \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_BIN_DIR) $(GOLANGCI_LINT_VERSION) \
		)

.PHONY: lint-deps
lint-deps: $(GOFUMPT)
lint-deps: $(GOLANGCI_LINT)

.PHONY: clean-tools
clean-tools:
	@rm -rf $(GOFUMPT)
	@rm -rf $(GOLANGCI_LINT)
