VERSION=snapshot
GOLANG_VERSION=1.11.2

# generate documentation.
doc:
	static-docs --in build/docs --out docs --theme gotenberg --title Gotenberg --subtitle "A Docker-powered stateless API for converting HTML, Markdown and Office documents to PDF."

# gofmt and goimports all go files.
fmt:
	go fmt ./...

# run all linters.
lint:
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t thecodingmachine/gotenberg:lint -f build/lint/Dockerfile .
	docker run --rm -it -v "$(PWD):/lint" thecodingmachine/gotenberg:lint

# run all tests.
tests:
	docker build -t thecodingmachine/gotenberg:base -f build/base/Dockerfile .
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) -t thecodingmachine/gotenberg:tests -f build/tests/Dockerfile .
	docker run --rm -it -v "$(PWD):/tests" thecodingmachine/gotenberg:tests

# build Docker image.
image:
	docker build -t thecodingmachine/gotenberg:base -f build/base/Dockerfile .
	docker build --build-arg GOLANG_VERSION=$(GOLANG_VERSION) --build-arg VERSION=$(VERSION) -t thecodingmachine/gotenberg:$(VERSION) -f build/package/Dockerfile .

# start the API using previously built Docker image.
gotenberg:
	docker run -it --rm -p "3000:3000" thecodingmachine/gotenberg:$(VERSION)