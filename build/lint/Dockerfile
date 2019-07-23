FROM thecodingmachine/gotenberg:workspace

# |--------------------------------------------------------------------------
# | GolangCI-Lint
# |--------------------------------------------------------------------------
# |
# | Installs GolangCI-Lint, a linters Runner for Go. 5x faster 
# | than gometalinter.
# |

ARG GOLANGCI_LINT_VERSION

RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b /usr/local/bin v${GOLANGCI_LINT_VERSION} &&\
    golangci-lint --version

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

# Define our working directory outside of $GOPATH (we're using go modules).
USER gotenberg
WORKDIR /gotenberg/lint

# Copy our code source.
COPY --chown=gotenberg:gotenberg . .

# Install module dependencies.
RUN go mod download &&\
    go mod verify

CMD ["golangci-lint", "run" ,"--tests=false", "--enable-all", "--disable=dupl" ]