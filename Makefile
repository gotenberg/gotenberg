VERSION=snapshot

fmt:
	go fmt ./...

testing:
	docker build -t thecodingmachine/gotenberg:ci -f build/ci/Dockerfile .
	docker run --rm -e "VERSION=$(VERSION)" -v "$(PWD):/ci" thecodingmachine/gotenberg:ci

build-image:
	docker build -t thecodingmachine/gotenberg:$(VERSION) -f build/package/Dockerfile .

server:
	docker run --rm -p "3000:3000" thecodingmachine/gotenberg:$(VERSION) gotenberg server