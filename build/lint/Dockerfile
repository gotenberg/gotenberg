ARG GOLANG_VERSION

FROM golang:${GOLANG_VERSION}-stretch

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | GolangCI-Lint
# |--------------------------------------------------------------------------
# |
# | Installs GolangCI-Lint, a linters Runner for Go. 5x faster 
# | than gometalinter.
# |

ENV GOLANGCI_LINT_VERSION 1.12.3

RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION} &&\
    golangci-lint --version

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

# Define our workding outside of $GOPATH (we're using go modules).
WORKDIR /lint

# Copy our module dependencies definitions.
COPY go.mod .
COPY go.sum .

# Install module dependencies.
RUN go mod download

CMD ["golangci-lint", "run" ,"--tests=false", "--enable-all", "--disable=dupl", "--disable=lll", "--disable=errcheck", "--disable=gosec", "--disable=gochecknoglobals", "--disable=gochecknoinits" ]