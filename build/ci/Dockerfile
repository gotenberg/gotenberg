FROM golang:1.11-stretch

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | GolangCI-Lint
# |--------------------------------------------------------------------------
# |
# | Installs GolangCI-Lint, a linters Runner for Go. 5x faster 
# | than gometalinter.
# |

RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh

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

# Installs project dependencies.
RUN go get

ENTRYPOINT ["build/ci/docker-entrypoint.sh"]