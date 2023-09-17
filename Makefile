SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Using directory as project name.
PROJECT_NAME := $(shell basename $(PROJECT_ROOT))
PROJECT_MODULE := $(shell go list -m)

default: help

ifeq ($(CI),true)
$(info Running in a CI environment, verbose mode is disabled)
else
VERBOSE="true"
endif

include tools/tools.mk

# include per-user customization after all variables are defined
-include Makefile.local

# Only for CI compliance
.PHONY: bootstrap
bootstrap: lint-deps # Install all dependencies

.PHONY: tidy
tidy: TIDY_CMD=go mod tidy
tidy:
	@$(TIDY_CMD)

.PHONY: gofmt
gofmt: lint-deps
gofmt: ## Format Go code
	@$(GOFUMPT) -extra -l -w .

.PHONY: lint
lint: lint-deps
lint: ## Lint the source code
	@echo "==> Linting source code..."
	@$(GOLANGCI_LINT) run --config=.golangci.yml --concurrency 1 --fix

	@echo "==> Checking Go mod..."
	@$(MAKE) tidy
	@if (git status --porcelain | grep -Eq "go\.(mod|sum)"); then \
		echo go.mod or go.sum needs updating; \
		git --no-pager diff go.mod; \
		git --no-pager diff go.sum; \
		exit 1; fi

.PHONY: compile
compile: # Compiles the packages but discards the resulting object, serving only as a check that the packages can be built
	CGO_ENABLED=0 go build -o /dev/null ./...

.PHONY: install
install: install-$(PROJECT_NAME)
install: ## Compile and install the main packages

.PHONY: install-$(PROJECT_NAME)
install-$(PROJECT_NAME):
	@if [ -x "$$(command -v $(PROJECT_NAME))" ]; then \
		echo "$(PROJECT_NAME) is already installed, do you want to re-install it? [y/N] " && read ans; \
			if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]  ; then \
				go install .; \
			else \
				echo "aborting install"; \
			exit -1; \
		fi; \
	else \
		go install .; \
	fi;

.PHONY: clean
clean: ## Remove build artifacts
	@echo "==> Removing build artifacts..."
	@rm -f $(if $(VERBOSE),-v) "$(GOPATH)/bin/$(PROJECT_NAME)"

HELP_FORMAT="    \033[36m%-15s\033[0m %s\n"
.PHONY: help
help: ## Display this usage information
	@echo "Valid targets:"
	@echo $(MAKEFILE_LIST) | \
		xargs grep -E '^[^ ]+:.*?## .*$$' -h | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""

FORCE:
