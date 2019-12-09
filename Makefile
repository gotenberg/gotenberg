GOLANG_VERSION=1.13
VERSION=snapshot
DOCKER_USER=
DOCKER_PASSWORD=
DOCKER_REGISTRY=thecodingmachine
GOTENBERG_USER_GID=1001
GOTENBERG_USER_UID=1001
GOLANGCI_LINT_VERSION=1.20.1
CODE_COVERAGE=0
TINI_VERSION=0.18.0
MAXIMUM_WAIT_TIMEOUT=30.0
MAXIMUM_WAIT_DELAY=10.0
MAXIMUM_WEBHOOK_URL_TIMEOUT=30.0
DEFAULT_WAIT_TIMEOUT=10.0
DEFAULT_WEBHOOK_URL_TIMEOUT=10.0
DEFAULT_LISTEN_PORT=3000
DISABLE_GOOGLE_CHROME=0
DISABLE_UNOCONV=0
LOG_LEVEL=INFO
ROOT_PATH=/
DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE=1048576

# build the base Docker image.
base:
	docker build --build-arg GOTENBERG_USER_GID=$(GOTENBERG_USER_GID) --build-arg GOTENBERG_USER_UID=$(GOTENBERG_USER_UID) -t $(DOCKER_REGISTRY)/gotenberg:base -f build/base/Dockerfile .

# build the workspace Docker image.
workspace:
	make base
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t $(DOCKER_REGISTRY)/gotenberg:workspace -f build/workspace/Dockerfile . 

# gofmt and goimports all go files.
fmt:
	go fmt ./...
	go mod tidy

# run all linters.
lint:
	make workspace
	docker build --build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) -t $(DOCKER_REGISTRY)/gotenberg:lint -f build/lint/Dockerfile .
	docker run --rm $(DOCKER_REGISTRY)/gotenberg:lint

# run all tests.
tests:
	make workspace
	./scripts/tests.sh $(DOCKER_REGISTRY) $(CODE_COVERAGE)

# generate documentation.
doc:
	make workspace
	docker build -t $(DOCKER_REGISTRY)/gotenberg:docs -f build/docs/Dockerfile . 
	docker run --rm -it -v "$(PWD):/gotenberg/docs" $(DOCKER_REGISTRY)/gotenberg:docs

# build Gotenberg Docker image.
image:
	make workspace
	docker build --build-arg VERSION=$(VERSION) --build-arg TINI_VERSION=$(TINI_VERSION) -t $(DOCKER_REGISTRY)/gotenberg:$(VERSION) -f build/package/Dockerfile .

# start the API using previously built Docker image.
gotenberg:
	docker run -it --rm -e MAXIMUM_WAIT_TIMEOUT=$(MAXIMUM_WAIT_TIMEOUT) -e MAXIMUM_WAIT_DELAY=$(MAXIMUM_WAIT_DELAY) -e MAXIMUM_WEBHOOK_URL_TIMEOUT=$(MAXIMUM_WEBHOOK_URL_TIMEOUT) -e DEFAULT_WEBHOOK_URL_TIMEOUT=$(DEFAULT_WEBHOOK_URL_TIMEOUT) -e MAXIMUM_WEBHOOK_URL_TIMEOUT=$(MAXIMUM_WEBHOOK_URL_TIMEOUT) -e DEFAULT_LISTEN_PORT=$(DEFAULT_LISTEN_PORT) -e DISABLE_GOOGLE_CHROME=$(DISABLE_GOOGLE_CHROME) -e DISABLE_UNOCONV=$(DISABLE_UNOCONV) -e LOG_LEVEL=$(LOG_LEVEL) -e ROOT_PATH=$(ROOT_PATH) -e DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE=$(DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE)  -p "$(DEFAULT_LISTEN_PORT):$(DEFAULT_LISTEN_PORT)" $(DOCKER_REPOSITORY)/gotenberg:$(VERSION)

# publish Gotenberg images according to version.
publish:
	make workspace
	./scripts/publish.sh $(GOLANG_VERSION) $(TINI_VERSION) $(DOCKER_REGISTRY) $(VERSION) $(DOCKER_USER) $(DOCKER_PASSWORD)