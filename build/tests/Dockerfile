FROM thecodingmachine/gotenberg:workspace

# Define our workding outside of $GOPATH (we're using go modules).
USER gotenberg
WORKDIR /gotenberg/tests

# Install module dependencies.
COPY --chown=gotenberg:gotenberg go.mod go.sum ./

RUN go mod download &&\
    go mod verify

# Copy our code source.
COPY --chown=gotenberg:gotenberg . .

ENTRYPOINT [ "build/tests/docker-entrypoint.sh" ]