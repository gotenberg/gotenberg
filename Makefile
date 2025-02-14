include .env

.PHONY: help
help: ## Show the help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: it
it: build build-tests ## Initialize the development environment

.PHONY: build
build: ## Build the Gotenberg's Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOTENBERG_USER_GID=$(GOTENBERG_USER_GID) \
	--build-arg GOTENBERG_USER_UID=$(GOTENBERG_USER_UID) \
	--build-arg NOTO_COLOR_EMOJI_VERSION=$(NOTO_COLOR_EMOJI_VERSION) \
	--build-arg PDFTK_VERSION=$(PDFTK_VERSION) \
	--build-arg PDFCPU_VERSION=$(PDFCPU_VERSION) \
	-t $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION) \
	-f $(DOCKERFILE) $(DOCKER_BUILD_CONTEXT)

GOTENBERG_GRACEFUL_SHUTDOWN_DURATION=30s
API_PORT=3000
API_PORT_FROM_ENV=
API_BIND_IP=
API_START_TIMEOUT=30s
API_TIMEOUT=30s
API_BODY_LIMIT=
API_ROOT_PATH="/"
API_TRACE_HEADER=Gotenberg-Trace
API_ENABLE_BASIC_AUTH=false
GOTENBERG_API_BASIC_AUTH_USERNAME=
GOTENBERG_API_BASIC_AUTH_PASSWORD=
API-DOWNLOAD-FROM-ALLOW-LIST=
API-DOWNLOAD-FROM-DENY-LIST=
API-DOWNLOAD-FROM-FROM-MAX-RETRY=4
API-DISABLE-DOWNLOAD-FROM=false
API_DISABLE_HEALTH_CHECK_LOGGING=false
API_ENABLE_DEBUG_ROUTE=false
CHROMIUM_RESTART_AFTER=10
CHROMIUM_MAX_QUEUE_SIZE=0
CHROMIUM_AUTO_START=false
CHROMIUM_START_TIMEOUT=20s
CHROMIUM_INCOGNITO=false
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
PDFENGINES_ENGINES=
PDFENGINES_MERGE_ENGINES=qpdf,pdfcpu,pdftk
PDFENGINES_SPLIT_ENGINES=pdfcpu,qpdf,pdftk
PDFENGINES_FLATTEN_ENGINES=qpdf
PDFENGINES_CONVERT_ENGINES=libreoffice-pdfengine
PDFENGINES_READ_METADATA_ENGINES=exiftool
PDFENGINES_WRITE_METADATA_ENGINES=exiftool
PDFENGINES_DISABLE_ROUTES=false
PROMETHEUS_NAMESPACE=gotenberg
PROMETHEUS_COLLECT_INTERVAL=1s
PROMETHEUS_DISABLE_ROUTE_LOGGING=false
PROMETHEUS_DISABLE_COLLECT=false
WEBHOOK_ALLOW_LIST=
WEBHOOK_DENY_LIST=
WEBHOOK_ERROR_ALLOW_LIST=
WEBHOOK_ERROR_DENY_LIST=
WEBHOOK_MAX_RETRY=4
WEBHOOK_RETRY_MIN_WAIT=1s
WEBHOOK_RETRY_MAX_WAIT=30s
WEBHOOK_CLIENT_TIMEOUT=30s
WEBHOOK_DISABLE=false

.PHONY: run
run: ## Start a Gotenberg container
	docker run --rm -it \
	-p $(API_PORT):$(API_PORT) \
	-e GOTENBERG_API_BASIC_AUTH_USERNAME=$(GOTENBERG_API_BASIC_AUTH_USERNAME) \
	-e GOTENBERG_API_BASIC_AUTH_PASSWORD=$(GOTENBERG_API_BASIC_AUTH_PASSWORD) \
	$(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION) \
	gotenberg \
	--gotenberg-graceful-shutdown-duration=$(GOTENBERG_GRACEFUL_SHUTDOWN_DURATION) \
	--api-port=$(API_PORT) \
	--api-port-from-env=$(API_PORT_FROM_ENV) \
	--api-bind-ip=$(API_BIND_IP) \
	--api-start-timeout=$(API_START_TIMEOUT) \
	--api-timeout=$(API_TIMEOUT) \
	--api-body-limit="$(API_BODY_LIMIT)" \
	--api-root-path=$(API_ROOT_PATH) \
	--api-trace-header=$(API_TRACE_HEADER) \
	--api-enable-basic-auth=$(API_ENABLE_BASIC_AUTH) \
	--api-download-from-allow-list=$(API-DOWNLOAD-FROM-ALLOW-LIST) \
	--api-download-from-deny-list=$(API-DOWNLOAD-FROM-DENY-LIST) \
	--api-download-from-max-retry=$(API-DOWNLOAD-FROM-FROM-MAX-RETRY) \
	--api-disable-download-from=$(API-DISABLE-DOWNLOAD-FROM) \
	--api-disable-health-check-logging=$(API_DISABLE_HEALTH_CHECK_LOGGING) \
	--api-enable-debug-route=$(API_ENABLE_DEBUG_ROUTE) \
	--chromium-restart-after=$(CHROMIUM_RESTART_AFTER) \
	--chromium-auto-start=$(CHROMIUM_AUTO_START) \
	--chromium-max-queue-size=$(CHROMIUM_MAX_QUEUE_SIZE) \
	--chromium-start-timeout=$(CHROMIUM_START_TIMEOUT) \
	--chromium-incognito=$(CHROMIUM_INCOGNITO) \
	--chromium-allow-insecure-localhost=$(CHROMIUM_ALLOW_INSECURE_LOCALHOST) \
	--chromium-ignore-certificate-errors=$(CHROMIUM_IGNORE_CERTIFICATE_ERRORS) \
	--chromium-disable-web-security=$(CHROMIUM_DISABLE_WEB_SECURITY) \
	--chromium-allow-file-access-from-files=$(CHROMIUM_ALLOW_FILE_ACCESS_FROM_FILES) \
	--chromium-host-resolver-rules=$(CHROMIUM_HOST_RESOLVER_RULES) \
	--chromium-proxy-server=$(CHROMIUM_PROXY_SERVER) \
	--chromium-allow-list="$(CHROMIUM_ALLOW_LIST)" \
	--chromium-deny-list="$(CHROMIUM_DENY_LIST)" \
	--chromium-clear-cache=$(CHROMIUM_CLEAR_CACHE) \
	--chromium-clear-cookies=$(CHROMIUM_CLEAR_COOKIES) \
	--chromium-disable-javascript=$(CHROMIUM_DISABLE_JAVASCRIPT) \
	--chromium-disable-routes=$(CHROMIUM_DISABLE_ROUTES) \
	--libreoffice-restart-after=$(LIBREOFFICE_RESTART_AFTER) \
	--libreoffice-max-queue-size=$(LIBREOFFICE_MAX_QUEUE_SIZE) \
	--libreoffice-auto-start=$(LIBREOFFICE_AUTO_START) \
	--libreoffice-start-timeout=$(LIBREOFFICE_START_TIMEOUT) \
	--libreoffice-disable-routes=$(LIBREOFFICE_DISABLE_ROUTES) \
	--log-level=$(LOG_LEVEL) \
	--log-format=$(LOG_FORMAT) \
	--log-fields-prefix=$(LOG_FIELDS_PREFIX) \
	--pdfengines-engines=$(PDFENGINES_ENGINES) \
	--pdfengines-merge-engines=$(PDFENGINES_MERGE_ENGINES) \
	--pdfengines-split-engines=$(PDFENGINES_SPLIT_ENGINES) \
	--pdfengines-flatten-engines=$(PDFENGINES_FLATTEN_ENGINES) \
	--pdfengines-convert-engines=$(PDFENGINES_CONVERT_ENGINES) \
	--pdfengines-read-metadata-engines=$(PDFENGINES_READ_METADATA_ENGINES) \
	--pdfengines-write-metadata-engines=$(PDFENGINES_WRITE_METADATA_ENGINES) \
	--pdfengines-disable-routes=$(PDFENGINES_DISABLE_ROUTES) \
	--prometheus-namespace=$(PROMETHEUS_NAMESPACE) \
	--prometheus-collect-interval=$(PROMETHEUS_COLLECT_INTERVAL) \
	--prometheus-disable-route-logging=$(PROMETHEUS_DISABLE_ROUTE_LOGGING) \
	--prometheus-disable-collect=$(PROMETHEUS_DISABLE_COLLECT) \
	--webhook-allow-list="$(WEBHOOK_ALLOW_LIST)" \
	--webhook-deny-list="$(WEBHOOK_DENY_LIST)" \
	--webhook-error-allow-list=$(WEBHOOK_ERROR_ALLOW_LIST) \
	--webhook-error-deny-list=$(WEBHOOK_ERROR_DENY_LIST) \
	--webhook-max-retry=$(WEBHOOK_MAX_RETRY) \
	--webhook-retry-min-wait=$(WEBHOOK_RETRY_MIN_WAIT) \
	--webhook-retry-max-wait=$(WEBHOOK_RETRY_MAX_WAIT) \
	--webhook-client-timeout=$(WEBHOOK_CLIENT_TIMEOUT) \
	--webhook-disable=$(WEBHOOK_DISABLE)

.PHONY: build-tests
build-tests: ## Build the tests' Docker image
	docker build \
	--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	--build-arg DOCKER_REPOSITORY=$(DOCKER_REPOSITORY) \
	--build-arg GOTENBERG_VERSION=$(GOTENBERG_VERSION) \
	--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
	-t $(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION)-tests \
	-f test/Dockerfile .

.PHONY: tests
tests: ## Start the testing environment
	docker run --rm -it \
	-v $(PWD):/tests \
	$(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION)-tests \
	bash

.PHONY: tests-once
tests-once: ## Run the tests once (prefer the "tests" command while developing)
	docker run --rm  \
	-v $(PWD):/tests \
	$(DOCKER_REGISTRY)/$(DOCKER_REPOSITORY):$(GOTENBERG_VERSION)-tests \
	gotest

# go install mvdan.cc/gofumpt@latest
# go install github.com/daixiang0/gci@latest
.PHONY: fmt
fmt: ## Format the code and "optimize" the dependencies
	gofumpt -l -w .
	gci write -s standard -s default -s "prefix(github.com/gotenberg/gotenberg/v8)" --skip-generated --skip-vendor --custom-order .
	go mod tidy

# go install golang.org/x/tools/cmd/godoc@latest
.PHONY: godoc
godoc: ## Run a webserver with Gotenberg godoc
	$(info http://localhost:6060/pkg/github.com/gotenberg/gotenberg/v8)
	godoc -http=:6060
