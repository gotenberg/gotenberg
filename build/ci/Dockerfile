ARG VERSION

FROM thecodingmachine/gotenberg:${VERSION}

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | Common libraries
# |--------------------------------------------------------------------------
# |
# | Libraries used in the build process of this image.
# |

RUN apt-get install -y git gcc

# |--------------------------------------------------------------------------
# | Golang
# |--------------------------------------------------------------------------
# |
# | Installs Golang.
# |

ENV GOLANG_VERSION 1.11.2

RUN wget -O go.tgz https://golang.org/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz &&\
    tar -C /usr/local -xzf go.tgz &&\
	rm go.tgz &&\
    export PATH="/usr/local/go/bin:$PATH" &&\
    go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

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
# | Last instructions to setup CI.
# |

# Defines our workding outside of $GOPATH (we're using go modules).
WORKDIR /ci

# Copies our module dependencies definitions.
COPY go.mod .
COPY go.sum .

# Installs module dependencies.
RUN go mod download

ENTRYPOINT ["build/ci/docker-entrypoint.sh"]