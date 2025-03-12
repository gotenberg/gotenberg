ARG GOLANG_VERSION
ARG DOCKER_REGISTRY
ARG DOCKER_REPOSITORY
ARG GOTENBERG_VERSION
ARG GOLANGCI_LINT_VERSION

FROM golang:${GOLANG_VERSION:-1.24}-alpine AS golang

# We're extending the Gotenberg's Docker image because our code relies on external
# dependencies like Google Chrome, LibreOffice, etc.
FROM $DOCKER_REGISTRY/$DOCKER_REPOSITORY:$GOTENBERG_VERSION

USER root

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
ENV CGO_ENABLED=1

COPY --from=golang /usr/local/go /usr/local/go

RUN apk update -q &&\
    apk add -q --no-cache \
    bash \
    sudo \
    # gcc for cgo.
    g++ \
    gcc \
    gcompat \
    make \
    pkgconf \
    # shadow adds usermod command used in docker-entrypoint.sh
    shadow \
    &&\
    mkdir -p "$GOPATH/src" "$GOPATH/bin" &&\
    chmod -R 777 "$GOPATH" &&\
    addgroup -S sudo &&\
    adduser gotenberg sudo &&\
    echo '%sudo ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers &&\
    # We cannot use $PATH in the next command (print $PATH instead of the environment variable value).
    sed -i 's#/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin#/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/go/bin:/usr/local/go/bin#g' /etc/sudoers &&\
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $GOLANGCI_LINT_VERSION &&\
    go install mvdan.cc/gofumpt@latest &&\
    go install github.com/daixiang0/gci@latest

COPY ./test/docker-entrypoint.sh /docker-entrypoint.sh
COPY ./test/golint.sh /usr/bin/golint
COPY ./test/gotest.sh /usr/bin/gotest
COPY ./test/gotodos.sh /usr/bin/gotodos

# Pristine working directory.
WORKDIR /tests

ENTRYPOINT [ "/sbin/tini", "--", "/docker-entrypoint.sh" ]
