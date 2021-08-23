# Note: ARG instructions do not create additional layers.
# Instead, next layers will concatenate them.
ARG GOLANG_VERSION
# Note: we have to repeat ARG instructions in each build stage that uses them.
ARG GOTENBERG_VERSION

FROM golang:$GOLANG_VERSION AS builder

ENV CGO_ENABLED 0

# Define the working directory outside of $GOPATH (we're using go modules).
WORKDIR /home

# Install module dependencies.
COPY go.mod go.sum ./

RUN go mod download &&\
    go mod verify

# Copy the source code.
COPY cmd ./cmd
COPY pkg ./pkg

# Build the binary.
ARG GOTENBERG_VERSION

RUN go build -o gotenberg -ldflags "-X 'github.com/gotenberg/gotenberg/v7/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg/main.go

FROM debian:buster-slim

ARG GOTENBERG_VERSION

LABEL author="Julien Neuhart" \
      description="A Docker-powered stateless API for PDF files." \
      github="https://github.com/gotenberg/gotenberg" \
      version="$GOTENBERG_VERSION" \
      website="https://gotenberg.dev"

# Improve fonts subpixel hinting and smoothing.
# Credits:
# https://github.com/arachnys/athenapdf/issues/69.
# https://github.com/arachnys/athenapdf/commit/ba25a8d80a25d08d58865519c4cd8756dc9a336d.
COPY build/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# Simple wrapper around Java and PDFtk.
COPY build/pdftk.sh /usr/bin/pdftk

# Setup the Docker image.
ARG GOTENBERG_USER_GID
ARG GOTENBERG_USER_UID
ARG PDFTK_VERSION

RUN \
    # Create a non-root user.
    # All processes in the Docker container will run with this dedicated user.
    groupadd --gid "$GOTENBERG_USER_GID" gotenberg &&\
    useradd --uid "$GOTENBERG_USER_UID" --gid gotenberg --shell /bin/bash --home /home/gotenberg --no-create-home gotenberg &&\
    mkdir /home/gotenberg &&\
    chown gotenberg: /home/gotenberg &&\
    # Install dependencies required for the next instructions or debugging.
    # Note: procps for "top" command (useful when debugging processes).
    # Note: tini is a helper for reaping zombie processes.
    apt-get update -qq &&\
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends curl gnupg procps tini &&\
    # Install fonts.
    # Credits:
    # https://github.com/arachnys/athenapdf/blob/master/cli/Dockerfile.
    # https://help.accusoft.com/PrizmDoc/v12.1/HTML/Installing_Asian_Fonts_on_Ubuntu_and_Debian.html.
    curl -o ./ttf-mscorefonts-installer_3.8_all.deb http://httpredir.debian.org/debian/pool/contrib/m/msttcorefonts/ttf-mscorefonts-installer_3.8_all.deb &&\
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
    ./ttf-mscorefonts-installer_3.8_all.deb \
    culmus \
    fonts-beng \
    fonts-hosny-amiri \
    fonts-lklug-sinhala \
    fonts-lohit-guru \
    fonts-lohit-knda \
    fonts-samyak-gujr \
    fonts-samyak-mlym \
    fonts-samyak-taml \
    fonts-sarai \
    fonts-sil-abyssinica \
    fonts-sil-padauk \
    fonts-telu \
    fonts-thai-tlwg \
    ttf-wqy-zenhei \
    fonts-arphic-uming \
    fonts-ipafont-mincho \
    fonts-ipafont-gothic \
    fonts-unfonts-core \
    # LibreOffice recommends.
    fonts-crosextra-caladea \
    fonts-crosextra-carlito \
    fonts-dejavu \
    fonts-dejavu-extra \
    fonts-liberation \
    fonts-liberation2 \
    fonts-linuxlibertine \
    fonts-noto-core \
    fonts-noto-mono \
    fonts-noto-ui-core \
    fonts-sil-gentium \
    fonts-sil-gentium-basic &&\
    rm -f ./ttf-mscorefonts-installer_3.8_all.deb &&\
    echo "deb https://httpredir.debian.org/debian/ buster-backports main contrib non-free" >> /etc/apt/sources.list &&\
    apt-get update -qq &&\
    # Install Chromium and LibreOffice.
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends chromium &&\
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends -t buster-backports libreoffice &&\
    # Download unoconv (Python script).
    curl -Ls https://raw.githubusercontent.com/dagwieers/unoconv/master/unoconv -o /usr/bin/unoconv &&\
    chmod +x /usr/bin/unoconv &&\
    # unoconv will look for the Python binary, which has to be at version 3.
    ln -s /usr/bin/python3 /usr/bin/python &&\
    # Download PDFtk.
    # Credits: https://github.com/thecodingmachine/gotenberg/pull/273.
    curl -o /usr/bin/pdftk-all.jar "https://gitlab.com/pdftk-java/pdftk/-/jobs/$PDFTK_VERSION/artifacts/raw/build/libs/pdftk-all.jar" &&\
    chmod a+x /usr/bin/pdftk-all.jar &&\
    # See https://github.com/nextcloud/docker/issues/380.
    mkdir -p /usr/share/man/man1mkdir -p /usr/share/man/man1 &&\
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends default-jre-headless &&\
    # Cleanup.
    # Note: the Debian image does automatically a clean after each install thanks to a hook.
    # Therefore, there is no need for apt-get clean.
    # See https://stackoverflow.com/a/24417119/3248473.
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* &&\
    # Print versions of main dependencies.
    chromium --version &&\
    libreoffice --version &&\
    unoconv --version &&\
    pdftk --version

# Copy the Gotenberg binary from the builder stage.
COPY --from=builder /home/gotenberg /usr/bin/

# Environment variables required by modules or else.
ENV GC_EXCLUDE_SUBSTR "hsperfdata_root,hsperfdata_gotenberg"
ENV CHROMIUM_BIN_PATH /usr/bin/chromium
ENV UNOCONV_BIN_PATH /usr/bin/unoconv
ENV PDFTK_BIN_PATH /usr/bin/pdftk

USER gotenberg
WORKDIR /home/gotenberg

# Default API port.
EXPOSE 3000

ENTRYPOINT [ "/usr/bin/tini", "--" ]
CMD [ "gotenberg" ]