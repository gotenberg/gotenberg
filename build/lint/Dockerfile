ARG GOLANGCI_LINT_VERSION

FROM golangci/golangci-lint:v$GOLANGCI_LINT_VERSION

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

# Define our working directory outside of $GOPATH (we're using go modules).
WORKDIR /gotenberg/lint

# Copy our code source.
COPY . .

# Install module dependencies.
RUN go mod download &&\
    go mod verify

CMD ["golangci-lint", "run" ,"--tests=false", "--enable-all", "--disable=dupl", "--disable=funlen", "--disable=wsl", "--disable=gocognit", "--disable=gomnd", "--disable=goimports" ]