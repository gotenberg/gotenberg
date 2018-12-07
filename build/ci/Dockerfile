FROM thecodingmachine/gotenberg:baseci

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

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