# |--------------------------------------------------------------------------
# | Binary
# |--------------------------------------------------------------------------
# |
# | Buils Gotenberg binary.
# |

FROM thecodingmachine/gotenberg:workspace AS workspace

ARG VERSION

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

# Define our workding outside of $GOPATH (we're using go modules).
WORKDIR /gotenberg/package

# Install module dependencies.
COPY go.mod go.sum ./

RUN go mod download &&\
    go mod verify

# Copy our source code.
COPY internal ./internal
COPY cmd ./cmd

# Build our binary.
RUN go build -o gotenberg -ldflags "-X main.version=${VERSION}" cmd/gotenberg/main.go

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

FROM thecodingmachine/gotenberg:base

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

COPY --from=workspace /gotenberg/package/gotenberg /usr/local/bin/

USER gotenberg
WORKDIR /gotenberg

EXPOSE 3000
CMD [ "gotenberg" ]