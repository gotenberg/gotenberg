include .env

.PHONY: help
help: ## Show the help
	@grep -hE '^[A-Za-z0-9_ \-]*?:.*##.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the Gotenberg's Docker image (use TARGET=gotenberg-chromium or TARGET=gotenberg-libreoffice for variants)
	docker build \
	--target $(TARGET) \
	-t $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION) \
	-f $(DOCKERFILE) $(DOCKER_BUILD_CONTEXT)

TZ=UTC
GOTENBERG_HIDE_BANNER=false
GOTENBERG_GRACEFUL_SHUTDOWN_DURATION=30s
GOTENBERG_BUILD_DEBUG_DATA=true
API_PORT=3000
API_PORT_FROM_ENV=
API_BIND_IP=
API_TLS_CERT_FILE=
API_TLS_KEY_FILE=
API_START_TIMEOUT=30s
API_TIMEOUT=30s
API_BODY_LIMIT=
API_ROOT_PATH=/
API_CORRELATION_ID_HEADER=Gotenberg-Trace
API_ENABLE_BASIC_AUTH=false
GOTENBERG_API_BASIC_AUTH_USERNAME=
GOTENBERG_API_BASIC_AUTH_PASSWORD=
API_ENABLE_OIDC_AUTH=false
API_OIDC_ISSUER=
API_OIDC_AUDIENCE=
API_OIDC_JWKS_URL=
API_DOWNLOAD_FROM_ALLOW_LIST=
# Empty, like the flag default since 8.32.0. A textual deny-list cannot
# enumerate every way to write a private address, so *_DENY_PRIVATE_IPS is the
# control to reach for. Left false here so local testing can reach the host.
API_DOWNLOAD_FROM_DENY_LIST=
API_DOWNLOAD_FROM_DENY_PRIVATE_IPS=false
API_DOWNLOAD_FROM_DENY_PUBLIC_IPS=false
API_DOWNLOAD_FROM_ENABLE_ENVIRONMENT_PROXY=false
API_DOWNLOAD_FROM_MAX_RETRY=4
API_DOWNLOAD_FROM_MAX_CONCURRENCY=10
API_DOWNLOAD_FROM_MAX_ENTRIES=0
API_DISABLE_DOWNLOAD_FROM=false
API_DISABLE_HEALTH_CHECK_ROUTE_TELEMETRY=true
API_DISABLE_ROOT_ROUTE_TELEMETRY=true
API_DISABLE_DEBUG_ROUTE_TELEMETRY=true
API_DISABLE_VERSION_ROUTE_TELEMETRY=true
API_ENABLE_DEBUG_ROUTE=false
CHROMIUM_RESTART_AFTER=100
CHROMIUM_MAX_QUEUE_SIZE=0
CHROMIUM_IDLE_SHUTDOWN_TIMEOUT=0
CHROMIUM_MAX_CONCURRENCY=6
CHROMIUM_AUTO_START=false
CHROMIUM_START_TIMEOUT=20s
CHROMIUM_ALLOW_INSECURE_LOCALHOST=false
CHROMIUM_IGNORE_CERTIFICATE_ERRORS=false
CHROMIUM_DISABLE_WEB_SECURITY=false
CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES=false
CHROMIUM_HOST_RESOLVER_RULES=
CHROMIUM_PROXY_SERVER=
CHROMIUM_ENABLE_ENVIRONMENT_PROXY=false
CHROMIUM_ALLOW_LIST=
CHROMIUM_DENY_LIST=^file:(?!//\/tmp/).*
CHROMIUM_DENY_PRIVATE_IPS=false
CHROMIUM_DENY_PUBLIC_IPS=false
CHROMIUM_SHARED_BROWSER_CONTEXT=false
CHROMIUM_CLEAR_CACHE=false
CHROMIUM_CLEAR_COOKIES=false
CHROMIUM_DISABLE_JAVASCRIPT=false
CHROMIUM_DISABLE_ROUTES=false
LIBREOFFICE_RESTART_AFTER=10
LIBREOFFICE_MAX_QUEUE_SIZE=0
LIBREOFFICE_IDLE_SHUTDOWN_TIMEOUT=0
LIBREOFFICE_AUTO_START=false
LIBREOFFICE_START_TIMEOUT=20s
LIBREOFFICE_ALLOW_LIST=
LIBREOFFICE_DENY_LIST=
LIBREOFFICE_DENY_PRIVATE_IPS=false
LIBREOFFICE_DENY_PUBLIC_IPS=false
LIBREOFFICE_ENABLE_MACROS=false
LIBREOFFICE_ENABLE_ENVIRONMENT_PROXY=false
LIBREOFFICE_DISABLE_ROUTES=false
LOG_LEVEL=info
LOG_FIELDS_PREFIX=
LOG_STD_FORMAT=auto
LOG_STD_ENABLE_GCP_FIELDS=false
LOG_STD_LEVEL_CASE=lower
PDF_MAX_CONCURRENCY=1
PDF_MAX_STREAM_BYTES=
PDF_MAX_EMBEDDED_FILE_BYTES=
PDF_MAX_IMAGE_PIXELS=0
PDF_ALLOW_LIST=
PDF_DENY_LIST=
PDF_DENY_PRIVATE_IPS=false
PDF_DENY_PUBLIC_IPS=false
PDF_ENABLE_ENVIRONMENT_PROXY=false
PDF_DISABLE_ROUTES=false
OTEL_SERVICE_NAME=gotenberg
OTEL_TRACES_EXPORTER=none
OTEL_METRICS_EXPORTER=none
OTEL_LOGS_EXPORTER=none
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_INSECURE=true
WEBHOOK_ENABLE_SYNC_MODE=false
WEBHOOK_ALLOW_LIST=
# See the note on API_DOWNLOAD_FROM_DENY_LIST.
WEBHOOK_DENY_LIST=
WEBHOOK_DENY_PRIVATE_IPS=false
WEBHOOK_DENY_PUBLIC_IPS=false
WEBHOOK_ENABLE_ENVIRONMENT_PROXY=false
WEBHOOK_MAX_RETRY=4
WEBHOOK_RETRY_MIN_WAIT=1s
WEBHOOK_RETRY_MAX_WAIT=30s
WEBHOOK_CLIENT_TIMEOUT=30s
WEBHOOK_DISABLE=false

# Export all variables so they are available to Compose
export

.PHONY: run
run: ## Start a Gotenberg container via Compose
	docker compose up gotenberg

.PHONY: telemetry
telemetry: ## Start an OpenTelemetry collector and OpenObserve containers via Compose
	docker compose up otel-collector openobserve

.PHONY: down
down: ## Stop all containers
	docker compose down -v

.PHONY: test-unit
test-unit: ## Run unit tests
	go test -race ./...

PLATFORM=
# Variant image to test, such as chromium or libreoffice, tagged
# GOTENBERG_VERSION-VARIANT. Empty tests the full image.
VARIANT=
# Number of scenarios run at once; 0 runs one per CPU.
CONCURRENCY=0
# Available tags:
# chromium
# api-limits
# chromium-concurrent
# chromium-convert-html
# chromium-image-deduplication
# chromium-convert-markdown
# chromium-convert-url
# chromium-screenshot-html
# chromium-screenshot-markdown
# chromium-screenshot-url
# chromium-ssrf
# debug
# health
# libreoffice
# libreoffice-convert
# libreoffice-ssrf
# output-filename
# pdf
# pdf-pipe
# pdf-merge
# pdf-split
# pdf-pages
# pdf-rotate
# pdf-stamp
# pdf-metadata
# pdf-bookmarks
# pdf-form
# pdf-attach
# pdf-encrypt
# pdf-remediate
# factur-x
# pdf-sign
# pdf-optimize
# pdf-redact
# pdf-colors
# pdf-read
# root
# version
# webhook
# download-from
TAGS=

.PHONY: test-integration
test-integration: ## Run integration tests (automatically retries failed scenarios up to 3 times)
	go test -timeout 40m -tags=integration -v github.com/gotenberg/gotenberg/v9/test/integration -args \
	--gotenberg-docker-repository=$(DOCKER_REPOSITORY) \
	--gotenberg-version=$(GOTENBERG_VERSION) \
	--gotenberg-variant=$(VARIANT) \
 	--gotenberg-container-platform=$(PLATFORM) \
 	--concurrency=$(CONCURRENCY) \
 	--tags="$(TAGS)"

# golangci-lint runs at a pinned version, so a Homebrew upgrade cannot move the
# findings under the zero-warnings policy. An env var of the same name overrides
# it.
GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: ## Lint Golang codebase
	$(GOLANGCI_LINT) run

.PHONY: lint-prettier
lint-prettier: ## Lint non-Golang codebase
	npx prettier --check .

.PHONY: lint-todo
lint-todo: ## Find TODOs in Golang codebase
	$(GOLANGCI_LINT) run --no-config --default=none --enable godox

# TODO: restore a plain "go fix ./..." once the errorsastype modernizer stops
# rewriting this codebase into code that does not compile. Re-check by dropping
# the flag and running "make fmt && make lint". Removing the analyzer upstream
# makes go fix fail with "flag provided but not defined", so this cannot rot
# silently.
#   errorsastype rewrites errors.As to errors.AsType[T] without checking that T
#   satisfies error, which breaks on api.HttpError since it does not embed
#   error. Still broken as of Go 1.27.1.
.PHONY: fmt
fmt: ## Format Golang codebase and "optimize" the dependencies
	go fix -errorsastype=false ./...
	$(GOLANGCI_LINT) fmt
	go mod tidy

.PHONY: prettify
prettify: ## Format non-Golang codebase
	npx prettier --write .

# go install golang.org/x/tools/cmd/godoc@latest
.PHONY: godoc
godoc: ## Run a webserver with Gotenberg godoc
	$(info http://localhost:6060/pkg/github.com/gotenberg/gotenberg/v9)
	godoc -http=:6060
