FROM thecodingmachine/gotenberg:workspace

# Define our workding outside of $GOPATH (we're using go modules).
USER gotenberg
WORKDIR /gotenberg/tests

# Copy our code source.
COPY --chown=gotenberg:gotenberg . .

# Install module dependencies.
RUN go mod download &&\
    go mod verify

ENTRYPOINT [ "build/tests/docker-entrypoint.sh" ]