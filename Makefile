.PHONY: help
help: ## Show the help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: it
it: build build-tests ## Initialize the development environment

GOLANG_VERSION=1.17
DOCKER_REPOSITORY=gotenberg
GOTENBERG_VERSION=snapshot
GOTENBERG_USER_GID=1001
GOTENBERG_USER_UID=1001
NOTO_COLOR_EMOJI_VERSION=v2.028 # See https://github.com/googlefonts/noto-emoji/releases.
PDFTK_VERSION=1527259628 # See https://gitlab.com/pdftk-java/pdftk/-/releases - Binary package.
GOLANGCI_LINT_VERSION=v1.43.0 # See https://github.com/golangci/golangci-lint/releases.

.PHONY: build
build: ## Build the Gotenberg's Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOTENBERG_USER_GID=$(GOTENBERG_USER_GID) \
	--build-arg GOTENBERG_USER_UID=$(GOTENBERG_USER_UID) \
	--build-arg NOTO_COLOR_EMOJI_VERSION=$(NOTO_COLOR_EMOJI_VERSION) \
	--build-arg PDFTK_VERSION=$(PDFTK_VERSION) \
	-t $(DOCKER_REPOSITORY)/gotenberg:$(GOTENBERG_VERSION) \
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
CHROMIUM_INCOGNITO=false
CHROMIUM_IGNORE_CERTIFICATE_ERRORS=false
CHROMIUM_DISABLE_WEB_SECURITY=false
CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES=false
CHROMIUM_PROXY_SERVER=
CHROMIUM_ALLOW_LIST=
CHROMIUM_DENY_LIST="^file:///[^tmp].*"
CHROMIUM_DISABLE_JAVASCRIPT=false
CHROMIUM_DISABLE_ROUTES=false
LIBREOFFICE_DISABLES_ROUTES=false
LOG_LEVEL=info
LOG_FORMAT=auto
PDFENGINES_ENGINES=
PDFENGINES_DISABLE_ROUTES=false
PROMETHEUS_NAMESPACE=gotenberg
PROMETHEUS_COLLECT_INTERVAL=1s
PROMETHEUS_DISABLE_ROUTE_LOGGING=false
PROMETHEUS_DISABLE_COLLECT=false
UNOCONV_DISABLE_LISTENER=false
WEBHOOK_ALLOW_LIST=
WEBHOOK_DENY_LIST=
WEBHOOK_ERROR_ALLOW_LIST=
WEBHOOK_ERROR_DENY_LIST=
WEBHOOK_MAX_RETRY=4
WEBHOOK_RETRY_MIN_WAIT=1s
WEBHOOK_RETRY_MAX_WAIT=30s
WEBHOOK_DISABLE=false

.PHONY: run
run: ## Start a Gotenberg container
	docker run --rm -it \
	-p $(API_PORT):$(API_PORT) \
	$(DOCKER_REPOSITORY)/gotenberg:$(GOTENBERG_VERSION) \
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
	--chromium-incognito=$(CHROMIUM_INCOGNITO) \
	--chromium-ignore-certificate-errors=$(CHROMIUM_IGNORE_CERTIFICATE_ERRORS) \
	--chromium-disable-web-security=$(CHROMIUM_DISABLE_WEB_SECURITY) \
	--chromium-allow-file-access-from-files=$(CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES) \
	--chromium-proxy-server=$(CHROMIUM_PROXY_SERVER) \
	--chromium-allow-list=$(CHROMIUM_ALLOW_LIST) \
	--chromium-deny-list=$(CHROMIUM_DENY_LIST) \
	--chromium-disable-javascript=$(CHROMIUM_DISABLE_JAVASCRIPT) \
	--chromium-disable-routes=$(CHROMIUM_DISABLE_ROUTES) \
	--libreoffice-disable-routes=$(LIBREOFFICE_DISABLES_ROUTES) \
	--log-level=$(LOG_LEVEL) \
	--log-format=$(LOG_FORMAT) \
	--pdfengines-engines=$(PDFENGINES_ENGINES) \
	--pdfengines-disable-routes=$(PDFENGINES_DISABLE_ROUTES) \
	--prometheus-namespace=$(PROMETHEUS_NAMESPACE) \
	--prometheus-collect-interval=$(PROMETHEUS_COLLECT_INTERVAL) \
	--prometheus-disable-route-logging=$(PROMETHEUS_DISABLE_ROUTE_LOGGING) \
	--prometheus-disable-collect=$(PROMETHEUS_DISABLE_COLLECT) \
	--unoconv-disable-listener=$(UNOCONV_DISABLE_LISTENER) \
	--webhook-allow-list=$(WEBHOOK_ALLOW_LIST) \
	--webhook-deny-list=$(WEBHOOK_DENY_LIST) \
	--webhook-error-allow-list=$(WEBHOOK_ERROR_ALLOW_LIST) \
	--webhook-error-deny-list=$(WEBHOOK_ERROR_DENY_LIST) \
	--webhook-max-retry=$(WEBHOOK_MAX_RETRY) \
	--webhook-retry-min-wait=$(WEBHOOK_RETRY_MIN_WAIT) \
	--webhook-retry-max-wait=$(WEBHOOK_RETRY_MAX_WAIT) \
	--webhook-disable=$(WEBHOOK_DISABLE)

.PHONY: build-tests
build-tests: ## Build the tests' Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg DOCKER_REPOSITORY=$(DOCKER_REPOSITORY) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
	-t $(DOCKER_REPOSITORY)/gotenberg:$(GOTENBERG_VERSION)-tests \
	-f test/Dockerfile .

.PHONY: tests
tests: ## Start the testing environment
	docker run --rm -it \
	-v $(PWD):/tests \
	$(DOCKER_REPOSITORY)/gotenberg:$(GOTENBERG_VERSION)-tests \
	bash

.PHONY: tests-once
tests-once: ## Run the tests once (prefer the "tests" command while developing)
	docker run --rm  \
	-v $(PWD):/tests \
	$(DOCKER_REPOSITORY)/gotenberg:$(GOTENBERG_VERSION)-tests \
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
release: ## Build the Gotenberg's Docker image for many platforms, then push it to a Docker repository
	./scripts/release.sh \
 	$(GOLANG_VERSION) \
	$(GOTENBERG_VERSION) \
	$(GOTENBERG_USER_GID) \
	$(GOTENBERG_USER_UID) \
	$(PDFTK_VERSION) \
	$(DOCKER_REPOSITORY)
