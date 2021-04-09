NAME=spaghetti

SHELL := env VERSION=$(VERSION) $(SHELL)
VERSION ?= $(shell date -u +%Y%m%d.%H.%M.%S)


# COLORS
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
RESET  := $(shell tput -Txterm sgr0)


TARGET_MAX_CHAR_NUM=20


define colored
	@echo '${GREEN}$1${RESET}'
endef

## Show help
help:
	${call colored, help is running...}
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  ${YELLOW}%-$(TARGET_MAX_CHAR_NUM)s${RESET} ${GREEN}%s${RESET}\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

## vet project
vet:
	${call colored, vet is running...}
	./scripts/vet.sh
.PHONY: vet

## Compile executable
compile:
	${call colored, compile is running...}
	./scripts/compile.sh
.PHONY: compile

## Release
release:
	./scripts/release.sh
.PHONY: release

## Release local snapshot
release-local-snapshot:
	${call colored, release is running...}
	./scripts/release-local-snapshot.sh
.PHONY: release-local-snapshot

## Installs tools from vendor.
install-tools: sync-vendor
	./scripts/install-tools.sh
.PHONY: install-tools

## Sync vendor of root project and tools.
sync-vendor:
	./scripts/sync-vendor.sh
.PHONY: sync-vendor

## Run linting for build pipeline
lint-pipeline:
	./scripts/run-linters-pipeline.sh
.PHONY: lint-pipeline

## Fix imports sorting.
imports:
	${call colored, fix-imports is running...}
	./scripts/fix-imports.sh
.PHONY: imports

## Format code.
fmt:
	${call colored, fmt is running...}
	./scripts/fmt.sh
.PHONY: fmt

## Format code and sort imports.
format-project: fmt imports
.PHONY: format-project
