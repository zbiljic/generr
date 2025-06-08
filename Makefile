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

# include per-user customization after all variables are defined
-include Makefile.local

.PHONY: tools
tools:
	@command -v mise >/dev/null 2>&1 || { \
	  echo >&2 "Error: 'mise' not found in your PATH."; \
	  echo >&2 "Quick-install: 'curl https://mise.run | sh'"; \
	  echo >&2 "Full install instructions: https://mise.jdx.dev/installing-mise.html"; \
	  exit 1; \
	}

# Only for CI compliance
.PHONY: bootstrap
bootstrap: tools # Install all dependencies
	@mise install

.PHONY: tidy
tidy: TIDY_CMD=go mod tidy
tidy:
	@$(TIDY_CMD)

.PHONY: gofmt
gofmt: tools
gofmt: ## Format Go code
	@mise x -- gofumpt -extra -l -w .

.PHONY: lint
lint: tools
lint: ## Lint the source code
	@echo "==> Linting source code..."
	@mise x -- golangci-lint run --config=.golangci.yml --fix

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
