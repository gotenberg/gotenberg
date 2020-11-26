# |--------------------------------------------------------------------------
# | Binary
# |--------------------------------------------------------------------------
# |
# | Builds Gotenberg binary.
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
# | Generate Open API specs
# |--------------------------------------------------------------------------
# |
# | Convert from YAML to JSON
# |

FROM simplealpine/yaml2json AS openapi
COPY openapi.yaml /openapi.yaml
RUN yaml2json /openapi.yaml > /openapi.json


FROM thecodingmachine/gotenberg:base

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | Tini
# |--------------------------------------------------------------------------
# |
# | An helper for reaping zombie processes.
# |

ARG TINI_VERSION

ADD --chown=gotenberg https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini-static /tini
RUN chmod +x /tini
ENTRYPOINT [ "/tini", "--" ]

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

COPY --from=workspace /gotenberg/package/gotenberg /usr/local/bin/

USER gotenberg
WORKDIR /gotenberg

# Copy the HTTP server static content
COPY static ./static
COPY --from=openapi /openapi.json /static/

EXPOSE 3000
CMD [ "gotenberg" ]
