.PHONY: help
help: ## Show the help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: it
it: build build-tests ## Initialize the development environment

GOLANG_VERSION=1.16
DOCKER_REGISTRY=gotenberg
GOTENBERG_VERSION=snapshot
GOTENBERG_USER_GID=1001
GOTENBERG_USER_UID=1001
PDFTK_VERSION=1353200058 # See https://gitlab.com/pdftk-java/pdftk/-/releases - Binary package.
GOLANGCI_LINT_VERSION=v1.42.0 # See https://github.com/golangci/golangci-lint/releases.

.PHONY: build
build: ## Build the Gotenberg's Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOTENBERG_USER_GID=$(GOTENBERG_USER_GID) \
	--build-arg GOTENBERG_USER_UID=$(GOTENBERG_USER_UID) \
	--build-arg PDFTK_VERSION=$(PDFTK_VERSION) \
	-t $(DOCKER_REGISTRY)/gotenberg:$(GOTENBERG_VERSION) \
	-f build/Dockerfile .

GOTENBERG_GRACEFUL_SHUTDOWN_DURATION=30s
API_PORT=3000
API_PORT_FROM_ENV=
API_READ_TIMEOUT=30s
API_PROCESS_TIMEOUT=30s
API_WRITE_TIMEOUT=30s
API_ROOT_PATH=/
API_TRACE_HEADER=Gotenberg-Trace
API_DISABLE_HEALTH_CHECK_LOGGING=false
API_WEBHOOK_ALLOW_LIST=
API_WEBHOOK_DENY_LIST=
API_WEBHOOK_ERROR_ALLOW_LIST=
API_WEBHOOK_ERROR_DENY_LIST=
API_WEBHOOK_MAX_RETRY=4
API_WEBHOOK_RETRY_MIN_WAIT=1s
API_WEBHOOK_RETRY_MAX_WAIT=30s
API_DISABLE_WEBHOOK=false
CHROMIUM_USER_AGENT=
CHROMIUM_INCOGNITO=false
CHROMIUM_IGNORE_CERTIFICATE_ERRORS=false
CHROMIUM_ALLOW_LIST=
CHROMIUM_DENY_LIST="^file:///[^tmp].*"
CHROMIUM_DISABLE_ROUTES=false
LIBREOFFICE_DISABLES_ROUTES=false
LOG_LEVEL=info
LOG_FORMAT=auto
PDFENGINES_ENGINES=
PDFENGINES_DISABLE_ROUTES=false

.PHONY: run
run: ## Start a Gotenberg container
	docker run --rm -it \
	-p $(API_PORT):$(API_PORT) \
	$(DOCKER_REGISTRY)/gotenberg:$(GOTENBERG_VERSION) \
	gotenberg \
	--gotenberg-graceful-shutdown-duration=$(GOTENBERG_GRACEFUL_SHUTDOWN_DURATION) \
	--api-port=$(API_PORT) \
	--api-port-from-env=$(API_PORT_FROM_ENV) \
	--api-read-timeout=$(API_READ_TIMEOUT) \
	--api-process-timeout=$(API_PROCESS_TIMEOUT) \
	--api-write-timeout=$(API_WRITE_TIMEOUT) \
	--api-root-path=$(API_ROOT_PATH) \
	--api-trace-header=$(API_TRACE_HEADER) \
	--api-disable-health-check-logging=$(API_DISABLE_HEALTH_CHECK_LOGGING) \
	--api-webhook-allow-list=$(API_WEBHOOK_ALLOW_LIST) \
	--api-webhook-deny-list=$(API_WEBHOOK_DENY_LIST) \
	--api-webhook-error-allow-list=$(API_WEBHOOK_ERROR_ALLOW_LIST) \
	--api-webhook-error-deny-list=$(API_WEBHOOK_ERROR_DENY_LIST) \
	--api-webhook-max-retry=$(API_WEBHOOK_MAX_RETRY) \
	--api-webhook-retry-min-wait=$(API_WEBHOOK_RETRY_MIN_WAIT) \
	--api-webhook-retry-max-wait=$(API_WEBHOOK_RETRY_MAX_WAIT) \
	--api-disable-webhook=$(API_DISABLE_WEBHOOK) \
	--chromium-user-agent=$(CHROMIUM_USER_AGENT) \
	--chromium-incognito=$(CHROMIUM_INCOGNITO) \
	--chromium-ignore-certificate-errors=$(CHROMIUM_IGNORE_CERTIFICATE_ERRORS) \
	--chromium-allow-list=$(CHROMIUM_ALLOW_LIST) \
	--chromium-deny-list=$(CHROMIUM_DENY_LIST) \
	--chromium-disable-routes=$(CHROMIUM_DISABLE_ROUTES) \
	--libreoffice-disable-routes=$(LIBREOFFICE_DISABLES_ROUTES) \
	--log-level=$(LOG_LEVEL) \
	--log-format=$(LOG_FORMAT) \
	--pdfengines-engines=$(PDFENGINES_ENGINES) \
	--pdfengines-disable-routes=$(PDFENGINES_DISABLE_ROUTES)

.PHONY: build-tests
build-tests: ## Build the tests' Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
	-t $(DOCKER_REGISTRY)/gotenberg:$(GOTENBERG_VERSION)-tests \
	-f test/Dockerfile .

.PHONY: tests
tests: ## Start the testing environment
	docker run --rm -it \
	-v $(PWD):/tests \
	$(DOCKER_REGISTRY)/gotenberg:$(GOTENBERG_VERSION)-tests \
	bash

.PHONY: tests-once
tests-once: ## Run the tests once (prefer the "tests" command while developing)
	docker run --rm  \
	-v $(PWD):/tests \
	$(DOCKER_REGISTRY)/gotenberg:$(GOTENBERG_VERSION)-tests \
	gotest

.PHONY: fmt
fmt: ## Format the code and "optimize" the dependencies
	go fmt ./...
	go mod tidy

.PHONY: godoc
godoc: ## Run a webserver with Gotenberg godoc (go get golang.org/x/tools/cmd/godoc)
	$(info http://localhost:6060/pkg/github.com/gotenberg/gotenberg/v7)
	godoc -http=:6060

.PHONY: release
release: ## Build the Gotenberg's Docker image for linux/amd64 and linux/arm64 platforms, then push it to a Docker Registry
	./scripts/release.sh \
 	$(GOLANG_VERSION) \
	$(GOTENBERG_VERSION) \
	$(GOTENBERG_USER_GID) \
	$(GOTENBERG_USER_UID) \
	$(PDFTK_VERSION) \
	$(DOCKER_REGISTRY)
