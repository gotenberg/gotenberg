VERSION=snapshot
GOLANG_VERSION=1.11.2

# gofmt and goimports all go files.
fmt:
	go fmt ./...

# Prepare all base images.
prepare:
	docker build -t thecodingmachine/gotenberg:base -f build/base/package/Dockerfile .
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t thecodingmachine/gotenberg:baseci -f build/base/ci/Dockerfile .

# Run all linters and tests.
testing:
	docker build -t thecodingmachine/gotenberg:ci -f build/ci/Dockerfile .
	docker run --rm -e "VERSION=$(VERSION)" -v "$(PWD):/ci" thecodingmachine/gotenberg:ci

# Build Gotenberg and Docker image.
build-image:
	docker run -it --rm -e GOOS=linux -e GOARCH=amd64 -e CGO_ENABLED=0 -v "$(PWD):/gotenberg" -w /gotenberg golang:$(GOLANG_VERSION)-stretch go build -o build/package/gotenberg -ldflags "-X main.version=${VERSION}" cmd/gotenberg/main.go
	docker build -t thecodingmachine/gotenberg:$(VERSION) -f build/package/Dockerfile .

# Start the API using previously built Docker image.
gotenberg:
	docker run -it --rm -p "3000:3000" thecodingmachine/gotenberg:$(VERSION)