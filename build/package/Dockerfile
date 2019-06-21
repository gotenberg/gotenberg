ARG GOLANG_VERSION

# |--------------------------------------------------------------------------
# | Binary
# |--------------------------------------------------------------------------
# |
# | Buils Gotenberg binary.
# |

FROM golang:${GOLANG_VERSION}-stretch AS golang

ARG VERSION

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

# Define our workding outside of $GOPATH (we're using go modules).
WORKDIR /gotenberg

# Copy our source code.
COPY internal ./internal
COPY cmd ./cmd
COPY go.sum go.sum
COPY go.mod go.mod

# Build our binary.
RUN go build -o /gotenberg/gotenberg -ldflags "-X main.version=${VERSION}" cmd/gotenberg/main.go

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

FROM thecodingmachine/gotenberg:base

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

COPY --from=golang /gotenberg/gotenberg /usr/local/bin/

ENV PM2_HOME=/gotenberg/.pm2
USER gotenberg
WORKDIR /gotenberg

EXPOSE 3000
CMD [ "gotenberg" ]