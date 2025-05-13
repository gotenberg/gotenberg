# ARG instructions do not create additional layers. Instead, next layers will
# concatenate them. Also, we have to repeat ARG instructions in each build
# stage that uses them.
ARG GOLANG_VERSION=1.24
ARG ALPINE_VERSION=3.21

# ----------------------------------------------
# Golang base stage
# ----------------------------------------------
FROM golang:$GOLANG_VERSION-alpine AS go-base

RUN \
    apk update -q &&\
    apk add -q --no-cache curl=8.12.1-r1

# ----------------------------------------------
# pdfcpu binary build stage
# ----------------------------------------------
# Note: this stage is required as pdfcpu does not release an armhf variant by
# default.

FROM go-base AS pdfcpu-binary-stage

ARG PDFCPU_VERSION=v0.8.1
ENV CGO_ENABLED=0

# Define the working directory outside of $GOPATH (we're using go modules).
WORKDIR /home

RUN curl -Ls "https://github.com/pdfcpu/pdfcpu/archive/refs/tags/$PDFCPU_VERSION.tar.gz" -o pdfcpu.tar.gz &&\
    tar --strip-components=1 -xvzf pdfcpu.tar.gz

# Install module dependencies.
RUN go mod download &&\
    go mod verify

RUN go build -o pdfcpu -ldflags "-s -w -X 'main.version=$PDFCPU_VERSION' -X 'github.com/pdfcpu/pdfcpu/pkg/pdfcpu.VersionStr=$PDFCPU_VERSION' -X main.builtBy=gotenberg" ./cmd/pdfcpu &&\
    # Verify installation.
    ./pdfcpu version

# ----------------------------------------------
# Gotenberg binary build stage
# ----------------------------------------------
FROM go-base AS gotenberg-binary-stage

ARG GOTENBERG_VERSION
ENV CGO_ENABLED=0

# Define the working directory outside of $GOPATH (we're using go modules).
WORKDIR /home

# Install module dependencies.
COPY go.mod go.sum ./

RUN go mod download &&\
    go mod verify

# Copy the source code.
COPY cmd ./cmd
COPY pkg ./pkg

RUN go build -o gotenberg -ldflags "-X 'github.com/gotenberg/gotenberg/v8/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg/main.go

# ----------------------------------------------
# Fonts stage
# ----------------------------------------------
FROM debian:12-slim AS fonts-stage

ARG NOTO_COLOR_EMOJI_VERSION=v2.047

RUN \
    # Install fonts.
    # Credits:
    # https://github.com/arachnys/athenapdf/blob/master/cli/Dockerfile.
    apt-get update -qq &&\
    apt-get upgrade -yqq &&\
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends curl ca-certificates \
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
    fonts-arphic-ukai \
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
    fonts-noto-cjk \
    fonts-noto-core \
    fonts-noto-mono \
    fonts-noto-ui-core \
    fonts-sil-gentium \
    fonts-sil-gentium-basic &&\
    # Add Color and Black-and-White Noto emoji font.
    # Credits:
    # https://github.com/gotenberg/gotenberg/pull/325.
    # https://github.com/googlefonts/noto-emoji.
    curl -Ls "https://github.com/googlefonts/noto-emoji/raw/$NOTO_COLOR_EMOJI_VERSION/fonts/NotoColorEmoji.ttf" -o /usr/local/share/fonts/NotoColorEmoji.ttf &&\
    # Cleanup.
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# ----------------------------------------------
# Final stage
# ----------------------------------------------
FROM alpine:$ALPINE_VERSION

ARG GOTENBERG_VERSION
ARG GOTENBERG_USER_GID=1001
ARG GOTENBERG_USER_UID=1001
ARG PDFTK_VERSION=v3.3.3

LABEL org.opencontainers.image.title="Gotenberg" \
    org.opencontainers.image.description="A containerized API for seamless PDF conversion." \
    org.opencontainers.image.version="$GOTENBERG_VERSION" \
    org.opencontainers.image.authors="Julien Neuhart <neuhart.julien@gmail.com>" \
    org.opencontainers.image.documentation="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

# Create a non-root user.
# All processes in the Docker container will run with this dedicated user.
RUN \
    addgroup -g "$GOTENBERG_USER_GID" gotenberg &&\
    adduser -D -u "$GOTENBERG_USER_UID" -G gotenberg -s /bin/bash -h /home/gotenberg gotenberg

# Install system dependencies required for the next instructions or debugging.
# Note: tini is a helper for reaping zombie processes.
RUN \
    apk update -q &&\
    apk add -q --no-cache curl=8.12.1-r1 gnupg=2.4.7-r0 tini=0.19.0-r3 poppler=24.02.0-r2 python3=3.12.10-r0 openjdk17-jre-headless=17.0.15_p6-r0

# Install Chromium.
RUN \
    apk update -q &&\
    apk add -q --no-cache chromium &&\
    # Verify installation.
    chromium --version

# Install LibreOffice & unoconverter.
RUN \
    apk update -q  &&\
    apk add -q --no-cache \
    --repository https://dl-cdn.alpinelinux.org/alpine/edge/main \
    --repository https://dl-cdn.alpinelinux.org/alpine/edge/community \
    libreoffice &&\
    apk add -q --no-cache py3-setuptools py3-looseversion &&\
    curl -Ls https://raw.githubusercontent.com/gotenberg/unoconverter/v0.1.1/unoconv -o /usr/bin/unoconverter &&\
    chmod +x /usr/bin/unoconverter &&\
    # Verify installations.
    libreoffice --version &&\
    unoconverter --version

# Install PDFtk, QPDF & ExifTool (PDF engines).
# See https://github.com/gotenberg/gotenberg/pull/273.
RUN \
    curl -o /usr/bin/pdftk-all.jar "https://gitlab.com/api/v4/projects/5024297/packages/generic/pdftk-java/$PDFTK_VERSION/pdftk-all.jar" &&\
    echo -e '#!/bin/sh\n\nexec java -jar /usr/bin/pdftk-all.jar "$@"' > /usr/bin/pdftk && \
    chmod +x /usr/bin/pdftk &&\
    apk update -q &&\
    apk add -q --no-cache qpdf exiftool &&\
    # Verify installations.
    pdftk --version &&\
    qpdf --version &&\
    exiftool --version

# Install fonts.
COPY --from=fonts-stage /usr/share/fonts /usr/share/fonts
COPY --from=fonts-stage /usr/local/share/fonts /usr/local/share/fonts

RUN \
    apk update -q &&\
    apk add -q --no-cache fontconfig &&\
    fc-cache -fv &&\
    fc-list

# Improve fonts subpixel hinting and smoothing.
# Credits:
# https://github.com/arachnys/athenapdf/issues/69.
# https://github.com/arachnys/athenapdf/commit/ba25a8d80a25d08d58865519c4cd8756dc9a336d.
COPY build/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# Copy the pdfcpu binary from the pdfcpu-binary-stage.
COPY --from=pdfcpu-binary-stage /home/pdfcpu /usr/bin/

# Copy the Gotenberg binary from the gotenberg-binary-stage.
COPY --from=gotenberg-binary-stage /home/gotenberg /usr/bin/

# Environment variables required by modules or else.
ENV CHROMIUM_BIN_PATH=/usr/bin/chromium
ENV LIBREOFFICE_BIN_PATH=/usr/lib/libreoffice/program/soffice.bin
ENV UNOCONVERTER_BIN_PATH=/usr/bin/unoconverter
ENV PDFTK_BIN_PATH=/usr/bin/pdftk
ENV QPDF_BIN_PATH=/usr/bin/qpdf
ENV EXIFTOOL_BIN_PATH=/usr/bin/exiftool
ENV PDFCPU_BIN_PATH=/usr/bin/pdfcpu

USER gotenberg
WORKDIR /home/gotenberg

# Default API port.
EXPOSE 3000

ENTRYPOINT [ "/sbin/tini", "--" ]
CMD [ "gotenberg" ]