include .env

.PHONY: help
help: ## Show the help
	@grep -hE '^[A-Za-z0-9_ \-]*?:.*##.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the Gotenberg's Docker image
	docker build \
	-t $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION) \
	-f $(DOCKERFILE) $(DOCKER_BUILD_CONTEXT)

GOTENBERG_HIDE_BANNER=false
GOTENBERG_GRACEFUL_SHUTDOWN_DURATION=30s
GOTENBERG_BUILD_DEBUG_DATA=true
API_PORT=3000
API_PORT_FROM_ENV=
API_BIND_IP=
API_START_TIMEOUT=30s
API_TIMEOUT=30s
API_BODY_LIMIT=
API_ROOT_PATH=/
API_TRACE_HEADER=Gotenberg-Trace
API_ENABLE_BASIC_AUTH=false
GOTENBERG_API_BASIC_AUTH_USERNAME=
GOTENBERG_API_BASIC_AUTH_PASSWORD=
API_DOWNLOAD_FROM_ALLOW_LIST=
API_DOWNLOAD_FROM_DENY_LIST=
API_DOWNLOAD_FROM_FROM_MAX_RETRY=4
API_DISABLE_DOWNLOAD_FROM=false
API_DISABLE_HEALTH_CHECK_LOGGING=false
API_ENABLE_DEBUG_ROUTE=false
CHROMIUM_RESTART_AFTER=10
CHROMIUM_MAX_QUEUE_SIZE=0
CHROMIUM_AUTO_START=false
CHROMIUM_START_TIMEOUT=20s
CHROMIUM_ALLOW_INSECURE_LOCALHOST=false
CHROMIUM_IGNORE_CERTIFICATE_ERRORS=false
CHROMIUM_DISABLE_WEB_SECURITY=false
CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES=false
CHROMIUM_HOST_RESOLVER_RULES=
CHROMIUM_PROXY_SERVER=
CHROMIUM_ALLOW_LIST=
CHROMIUM_DENY_LIST=^file:(?!//\/tmp/).*
CHROMIUM_CLEAR_CACHE=false
CHROMIUM_CLEAR_COOKIES=false
CHROMIUM_DISABLE_JAVASCRIPT=false
CHROMIUM_DISABLE_ROUTES=false
LIBREOFFICE_RESTART_AFTER=10
LIBREOFFICE_MAX_QUEUE_SIZE=0
LIBREOFFICE_AUTO_START=false
LIBREOFFICE_START_TIMEOUT=20s
LIBREOFFICE_DISABLE_ROUTES=false
LOG_LEVEL=info
LOG_FORMAT=auto
LOG_FIELDS_PREFIX=
LOG_ENABLE_GCP_FIELDS=false
PDFENGINES_MERGE_ENGINES=qpdf,pdfcpu,pdftk
PDFENGINES_SPLIT_ENGINES=pdfcpu,qpdf,pdftk
PDFENGINES_FLATTEN_ENGINES=qpdf
PDFENGINES_CONVERT_ENGINES=libreoffice-pdfengine
PDFENGINES_READ_METADATA_ENGINES=exiftool
PDFENGINES_WRITE_METADATA_ENGINES=exiftool
PDFENGINES_ENCRYPT_ENGINES=qpdf,pdfcpu,pdftk
PDFENGINES_DISABLE_ROUTES=false
PDFENGINES_EMBED_ENGINES=pdfcpu
PROMETHEUS_NAMESPACE=gotenberg
PROMETHEUS_COLLECT_INTERVAL=5s
PROMETHEUS_DISABLE_ROUTE_LOGGING=false
PROMETHEUS_DISABLE_COLLECT=false
OTEL_SERVICE_NAME=gotenberg
OTEL_LOG_EXPORTER_PROTOCOL=grpc
OTEL_ENABLE_LOG_EXPORTER=false
OTEL_METRIC_EXPORTER_PROTOCOL=grpc
OTEL_ENABLE_METRIC_EXPORTER=false
OTEL_METRICS_COLLECT_INTERVAL=5s
OTEL_SPAN_EXPORTER_PROTOCOL=grpc
OTEL_ENABLE_SPAN_EXPORTER=false
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317
OTEL_EXPORTER_OTLP_INSECURE=true
WEBHOOK_ENABLE_SYNC_MODE=false
WEBHOOK_ALLOW_LIST=
WEBHOOK_DENY_LIST=
WEBHOOK_ERROR_ALLOW_LIST=
WEBHOOK_ERROR_DENY_LIST=
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

.PHONY: collector
collector: ## Start a Jaeger collector via Compose
	docker compose up jaeger

.PHONY: down
down: ## Stop all containers
	docker compose down -v

.PHONY: test-unit
test-unit: ## Run unit tests
	go test -race ./...

PLATFORM=
NO_CONCURRENCY=false
# Available tags:
# chromium
# chromium-convert-html
# chromium-convert-markdown
# chromium-convert-url
# debug
# health
# libreoffice
# libreoffice-convert
# output-filename
# pdfengines
# pdfengines-convert
# pdfengines-embed
# embed
# pdfengines-encrypt
# encrypt
# pdfengines-flatten
# flatten
# pdfengines-merge
# merge
# pdfengines-metadata
# metadata
# pdfengines-split
# split
# prometheus-metrics
# root
# version
# webhook
# download-from
TAGS=

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -timeout 40m -tags=integration -v github.com/gotenberg/gotenberg/v8/test/integration -args \
	--gotenberg-docker-repository=$(DOCKER_REPOSITORY) \
	--gotenberg-version=$(GOTENBERG_VERSION) \
 	--gotenberg-container-platform=$(PLATFORM) \
 	--no-concurrency=$(NO_CONCURRENCY) \
 	--tags="$(TAGS)"

.PHONY: lint
lint: ## Lint Golang codebase
	golangci-lint run

.PHONY: lint-prettier
lint-prettier: ## Lint non-Golang codebase
	npx prettier --check .

.PHONY: lint-todo
lint-todo: ## Find TODOs in Golang codebase
	golangci-lint run --no-config --disable-all --enable godox

.PHONY: fmt
fmt: ## Format Golang codebase and "optimize" the dependencies
	golangci-lint fmt
	go mod tidy

.PHONY: prettify
prettify: ## Format non-Golang codebase
	npx prettier --write .

# go install golang.org/x/tools/cmd/godoc@latest
.PHONY: godoc
godoc: ## Run a webserver with Gotenberg godoc
	$(info http://localhost:6060/pkg/github.com/gotenberg/gotenberg/v8)
	godoc -http=:6060
