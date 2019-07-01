GOLANG_VERSION=1.12
VERSION=snapshot
DOCKER_USER=
DOCKER_PASSWORD=
DOCKER_REPOSITORY=thecodingmachine
DEFAULT_WAIT_TIMEOUT=10
DEFAULT_LISTEN_PORT=3000
DISABLE_GOOGLE_CHROME=0
DISABLE_UNOCONV=0
LOG_LEVEL=INFO

# generate documentation.
doc:
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t $(DOCKER_REPOSITORY)/gotenberg:docs -f build/docs/Dockerfile . 
	docker run --rm -it -v "$(PWD):/docs" $(DOCKER_REPOSITORY)/gotenberg:docs

# gofmt and goimports all go files.
fmt:
	go fmt ./...
	go mod tidy

# run all linters.
lint:
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t $(DOCKER_REPOSITORY)/gotenberg:lint -f build/lint/Dockerfile .
	docker run --rm -it -v "$(PWD):/lint" $(DOCKER_REPOSITORY)/gotenberg:lint

# run all tests.
tests:
	docker build -t $(DOCKER_REPOSITORY)/gotenberg:base -f build/base/Dockerfile .
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t $(DOCKER_REPOSITORY)/gotenberg:tests -f build/tests/Dockerfile .
	docker run --rm -it -v "$(PWD):/tests" $(DOCKER_REPOSITORY)/gotenberg:tests

# build Docker image.
image:
	docker build -t $(DOCKER_REPOSITORY)/gotenberg:base -f build/base/Dockerfile .
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) --build-arg VERSION=$(VERSION) -t $(DOCKER_REPOSITORY)/gotenberg:$(VERSION) -f build/package/Dockerfile .

# start the API using previously built Docker image.
gotenberg:
	docker run -it --rm -e DEFAULT_WAIT_TIMEOUT=$(DEFAULT_WAIT_TIMEOUT) -e DEFAULT_LISTEN_PORT=$(DEFAULT_LISTEN_PORT) -e DISABLE_GOOGLE_CHROME=$(DISABLE_GOOGLE_CHROME) -e DISABLE_UNOCONV=$(DISABLE_UNOCONV) -e LOG_LEVEL=$(LOG_LEVEL)  -p "3000:$(DEFAULT_LISTEN_PORT)" $(DOCKER_REPOSITORY)/gotenberg:$(VERSION)

# publish Gotenberg images according to version.
publish:
	./scripts/publish.sh $(GOLANG_VERSION) $(VERSION) $(DOCKER_USER) $(DOCKER_PASSWORD)