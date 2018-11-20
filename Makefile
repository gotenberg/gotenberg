VERSION=snapshot

# gofmt and goimports all go files.
fmt:
	go fmt ./...

# Run all linters and tests.
testing:
	docker build -t thecodingmachine/gotenberg:ci -f build/ci/Dockerfile .
	docker run --rm -e "VERSION=$(VERSION)" -v "$(PWD):/ci" thecodingmachine/gotenberg:ci

# Build Gotenberg and Docker image.
build-image:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/package/gotenberg -ldflags "-X main.version=${VERSION}" cmd/gotenberg/main.go
	docker build -t thecodingmachine/gotenberg:$(VERSION) -f build/package/Dockerfile .

# Start the API using previously built Docker image.
server:
	docker run -it --rm -p "3000:3000" thecodingmachine/gotenberg:$(VERSION) gotenberg server