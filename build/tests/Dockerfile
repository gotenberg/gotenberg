ARG GOLANG_VERSION

FROM golang:${GOLANG_VERSION}-stretch AS golang

FROM thecodingmachine/gotenberg:base

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

COPY --from=golang /usr/local/go /usr/local/go

RUN export PATH="/usr/local/go/bin:$PATH" &&\
    go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

# Define our workding outside of $GOPATH (we're using go modules).
WORKDIR /tests

# Copy our module dependencies definitions.
COPY go.mod .
COPY go.sum .

# Install module dependencies.
RUN go mod download

ENTRYPOINT [ "build/tests/docker-entrypoint.sh" ]