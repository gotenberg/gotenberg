# ARG instructions do not create additional layers. Instead, next layers will
# concatenate them. Also, we have to repeat ARG instructions in each build
# stage that uses them.
ARG GOLANG_VERSION=1.26.0

# ----------------------------------------------
# pdfcpu binary build stage
# ----------------------------------------------
# Note: this stage is required as pdfcpu does not release an armhf variant by
# default.
FROM golang:$GOLANG_VERSION AS pdfcpu-binary-stage

# See https://github.com/pdfcpu/pdfcpu/releases.
ARG PDFCPU_VERSION=v0.11.1
ENV CGO_ENABLED=0

# Define the working directory outside of $GOPATH (we're using go modules).
WORKDIR /home

RUN curl -Ls "https://github.com/pdfcpu/pdfcpu/archive/refs/tags/$PDFCPU_VERSION.tar.gz" -o pdfcpu.tar.gz \
    && tar --strip-components=1 -xvzf pdfcpu.tar.gz

# Install module dependencies.
RUN go mod download \
    && go mod verify

RUN go build -o pdfcpu -ldflags "-s -w -X 'main.version=$PDFCPU_VERSION' -X 'github.com/pdfcpu/pdfcpu/pkg/pdfcpu.VersionStr=$PDFCPU_VERSION' -X main.builtBy=gotenberg" ./cmd/pdfcpu \
    # Verify installation.
    && ./pdfcpu version

# ----------------------------------------------
# Gotenberg binary build stage
# ----------------------------------------------
FROM golang:$GOLANG_VERSION AS gotenberg-binary-stage

ARG GOTENBERG_VERSION=snapshot
ENV CGO_ENABLED=0

# Define the working directory outside of $GOPATH (we're using go modules).
WORKDIR /home

# Install module dependencies.
COPY go.mod go.sum ./

RUN go mod download \
    && go mod verify

# Copy the source code.
COPY cmd ./cmd
COPY pkg ./pkg

RUN go build -o gotenberg -ldflags "-s -w -X 'github.com/gotenberg/gotenberg/v8/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg/main.go \
    && go build -o gotenberg-chromium -ldflags "-s -w -X 'github.com/gotenberg/gotenberg/v8/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg-chromium/main.go \
    && go build -o gotenberg-libreoffice -ldflags "-s -w -X 'github.com/gotenberg/gotenberg/v8/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg-libreoffice/main.go

# ----------------------------------------------
# Custom JRE stage
# Credits: https://github.com/jodconverter/docker-image-jodconverter-runtime
# ----------------------------------------------
FROM debian:13-slim AS custom-jre-stage

RUN apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends default-jdk-headless binutils

# Note: jdeps helps finding which modules a JAR requires.
# Currently only for PDFtk, as we don't rely on LibreOffice UNO Java SDK.
ENV JAVA_MODULES=java.base,java.desktop,java.naming,java.sql

RUN jlink \
    --add-modules $JAVA_MODULES \
    --strip-debug \
    --no-man-pages \
    --no-header-files \
    --compress=2 \
    --output /custom-jre

# ----------------------------------------------
# Downloader stage
# ----------------------------------------------
FROM debian:13-slim AS downloader-stage

# See https://gitlab.com/pdftk-java/pdftk/-/releases - Binary package.
ARG PDFTK_VERSION=v3.3.3

RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends curl ca-certificates

WORKDIR /downloads

RUN curl -Ls https://raw.githubusercontent.com/gotenberg/unoconverter/v0.2.0/unoconv -o unoconverter \
    && chmod +x unoconverter

RUN curl -o pdftk-all.jar "https://gitlab.com/api/v4/projects/5024297/packages/generic/pdftk-java/$PDFTK_VERSION/pdftk-all.jar" \
    && chmod a+x pdftk-all.jar

# ----------------------------------------------
# Base image stage
# ----------------------------------------------
FROM debian:13-slim AS base-image-stage

ARG TIMEZONE=UTC
ENV TZ=$TIMEZONE

COPY --link --from=custom-jre-stage /custom-jre /opt/java

ENV PATH="/opt/java/bin:${PATH}"

# ----------------------------------------------
# Common stage
# ----------------------------------------------
FROM base-image-stage AS common-stage

ARG GOTENBERG_USER_GID=1001
ARG GOTENBERG_USER_UID=1001

# Create a non-root user.
# All processes in the Docker container will run with this dedicated user.
RUN groupadd --gid "$GOTENBERG_USER_GID" gotenberg \
    && useradd --uid "$GOTENBERG_USER_UID" --gid gotenberg --shell /bin/bash --home /home/gotenberg --no-create-home gotenberg \
    && mkdir /home/gotenberg \
    && chown gotenberg: /home/gotenberg

# Note: tini is a helper for reaping zombie processes.
RUN apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
        tini \
        # Install fonts.
        # Essential metric-compatible fonts for LibreOffice layout fidelity (replaces MS Fonts).
        fonts-crosextra-carlito \
        fonts-crosextra-caladea \
        fonts-liberation \
        fonts-liberation2 \
        # Reliable general-purpose fallback for Chromium.
        fonts-dejavu \
        # Unified CJK (Chinese, Japanese, Korean) support.
        fonts-noto-cjk \
        # Standard Emoji support.
        fonts-noto-color-emoji \
        # Tofu prevention.
        fonts-noto-core \
        # Install QPDF & ExifTool (PDF engines).
        qpdf exiftool \
    # See https://github.com/nextcloud/docker/issues/380.
    && mkdir -p /usr/share/man/man1 \
    # Cleanup.
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# PDFtk bash wrapper.
# See https://github.com/gotenberg/gotenberg/pull/273.
RUN printf '#!/bin/bash\n\nexec java -jar /usr/bin/pdftk-all.jar "$@"' > /usr/bin/pdftk \
    && chmod +x /usr/bin/pdftk

# Support for arbitrary user IDs (OpenShift).
# See:
# https://github.com/gotenberg/gotenberg/issues/1049.
# https://docs.redhat.com/en/documentation/openshift_container_platform/4.15/html/images/creating-images#use-uid_create-images.
RUN usermod -aG root gotenberg \
    && chgrp -R 0 /home/gotenberg \
    && chmod -R g=u /home/gotenberg

# COPY instructions last to maximize cache reuse when only source code changes.

# Improve fonts subpixel hinting and smoothing.
# Credits:
# https://github.com/arachnys/athenapdf/issues/69.
# https://github.com/arachnys/athenapdf/commit/ba25a8d80a25d08d58865519c4cd8756dc9a336d.
COPY --link build/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# Copy PDFtk jar.
COPY --link --from=downloader-stage /downloads/pdftk-all.jar /usr/bin/pdftk-all.jar

# Copy the pdfcpu binary (shared across all variants).
COPY --link --from=pdfcpu-binary-stage /home/pdfcpu /usr/bin/

# Set default characterset encoding to UTF-8.
# See:
# https://github.com/gotenberg/gotenberg/issues/104
# https://github.com/gotenberg/gotenberg/issues/730
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Environment variables for PDF engines.
ENV PDFTK_BIN_PATH=/usr/bin/pdftk
ENV QPDF_BIN_PATH=/usr/bin/qpdf
ENV EXIFTOOL_BIN_PATH=/usr/bin/exiftool
ENV PDFCPU_BIN_PATH=/usr/bin/pdfcpu

# OpenTelemetry defaults (noop - no telemetry overhead unless explicitly enabled).
ENV OTEL_TRACES_EXPORTER=none
ENV OTEL_METRICS_EXPORTER=none
ENV OTEL_LOGS_EXPORTER=none

# ----------------------------------------------
# Final stage: full (Chromium + LibreOffice)
# ----------------------------------------------
FROM common-stage AS gotenberg

ARG GOTENBERG_VERSION=snapshot
ARG GOTENBERG_USER_GID=1001
ARG GOTENBERG_USER_UID=1001

LABEL org.opencontainers.image.title="Gotenberg" \
    org.opencontainers.image.description="A Docker-based API for converting documents to PDF." \
    org.opencontainers.image.version="$GOTENBERG_VERSION" \
    org.opencontainers.image.authors="Julien Neuhart <neuhart.julien@gmail.com>" \
    org.opencontainers.image.documentation="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

# Install Chromium.
RUN apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends chromium \
    # Cleanup.
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Install LibreOffice & unoconverter.
RUN echo "deb http://deb.debian.org/debian trixie-backports main" >> /etc/apt/sources.list \
    && apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
        python3 python3-distutils-extra \
        # Install Hyphenation for LibreOffice.
        # Credits: https://wiki.archlinux.org/title/LibreOffice.
        hyphen-af hyphen-as hyphen-be hyphen-bg hyphen-bn hyphen-ca hyphen-cs hyphen-da hyphen-de hyphen-el \
        hyphen-en-gb hyphen-en-us hyphen-eo hyphen-es hyphen-fr hyphen-gl hyphen-gu hyphen-hi hyphen-hr hyphen-hu \
        hyphen-id hyphen-is hyphen-it hyphen-kn hyphen-lt hyphen-lv hyphen-ml hyphen-mn hyphen-mr hyphen-nl \
        hyphen-no hyphen-or hyphen-pa hyphen-pl hyphen-pt-br hyphen-pt-pt hyphen-ro hyphen-ru hyphen-sk hyphen-sl \
        hyphen-sr hyphen-sv hyphen-ta hyphen-te hyphen-th hyphen-uk hyphen-zu \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends -t trixie-backports \
        libreoffice-writer libreoffice-calc libreoffice-impress libreoffice-draw python3-uno \
    # unoconverter will look for the Python binary, which has to be at version 3.
    && ln -s /usr/bin/python3 /usr/bin/python \
    # Cleanup.
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# COPY instructions last to maximize cache reuse.

# Copy the Gotenberg binary (full variant with all modules).
COPY --link --from=gotenberg-binary-stage /home/gotenberg /usr/bin/

# Copy unoconverter.
COPY --link --from=downloader-stage /downloads/unoconverter /usr/bin/unoconverter

# Copy dictionnaries so that hyphens work on Chromium.
# See https://github.com/gotenberg/gotenberg/issues/1293.
COPY --link --chown="$GOTENBERG_USER_UID:$GOTENBERG_USER_GID" build/chromium-hyphen-data /opt/gotenberg/chromium-hyphen-data

ENV CHROMIUM_BIN_PATH=/usr/bin/chromium
ENV CHROMIUM_HYPHEN_DATA_DIR_PATH=/opt/gotenberg/chromium-hyphen-data
ENV LIBREOFFICE_BIN_PATH=/usr/lib/libreoffice/program/soffice.bin
ENV UNOCONVERTER_BIN_PATH=/usr/bin/unoconverter

USER gotenberg
WORKDIR /home/gotenberg

# Default API port.
EXPOSE 3000

ENTRYPOINT [ "/usr/bin/tini", "--" ]
CMD [ "gotenberg" ]

# ----------------------------------------------
# Final stage: Chromium only
# ----------------------------------------------
FROM common-stage AS gotenberg-chromium

ARG GOTENBERG_VERSION=snapshot
ARG GOTENBERG_USER_GID=1001
ARG GOTENBERG_USER_UID=1001

LABEL org.opencontainers.image.title="Gotenberg (Chromium)" \
    org.opencontainers.image.description="A Docker-based API for converting documents to PDF — Chromium variant." \
    org.opencontainers.image.version="$GOTENBERG_VERSION" \
    org.opencontainers.image.authors="Julien Neuhart <neuhart.julien@gmail.com>" \
    org.opencontainers.image.documentation="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

# Install Chromium.
RUN apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends chromium \
    # Cleanup.
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# COPY instructions last to maximize cache reuse.

# Copy the Gotenberg binary (Chromium variant, no LibreOffice modules).
COPY --link --from=gotenberg-binary-stage /home/gotenberg-chromium /usr/bin/gotenberg

# Copy dictionnaries so that hyphens work on Chromium.
# See https://github.com/gotenberg/gotenberg/issues/1293.
COPY --link --chown="$GOTENBERG_USER_UID:$GOTENBERG_USER_GID" build/chromium-hyphen-data /opt/gotenberg/chromium-hyphen-data

ENV CHROMIUM_BIN_PATH=/usr/bin/chromium
ENV CHROMIUM_HYPHEN_DATA_DIR_PATH=/opt/gotenberg/chromium-hyphen-data

USER gotenberg
WORKDIR /home/gotenberg

# Default API port.
EXPOSE 3000

ENTRYPOINT [ "/usr/bin/tini", "--" ]
CMD [ "gotenberg" ]

# ----------------------------------------------
# Final stage: LibreOffice only
# ----------------------------------------------
FROM common-stage AS gotenberg-libreoffice

ARG GOTENBERG_VERSION=snapshot

LABEL org.opencontainers.image.title="Gotenberg (LibreOffice)" \
    org.opencontainers.image.description="A Docker-based API for converting documents to PDF — LibreOffice variant." \
    org.opencontainers.image.version="$GOTENBERG_VERSION" \
    org.opencontainers.image.authors="Julien Neuhart <neuhart.julien@gmail.com>" \
    org.opencontainers.image.documentation="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

# Install LibreOffice & unoconverter.
RUN echo "deb http://deb.debian.org/debian trixie-backports main" >> /etc/apt/sources.list \
    && apt-get update -qq \
    && apt-get upgrade -yqq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
        python3 python3-distutils-extra \
        # Install Hyphenation for LibreOffice.
        # Credits: https://wiki.archlinux.org/title/LibreOffice.
        hyphen-af hyphen-as hyphen-be hyphen-bg hyphen-bn hyphen-ca hyphen-cs hyphen-da hyphen-de hyphen-el \
        hyphen-en-gb hyphen-en-us hyphen-eo hyphen-es hyphen-fr hyphen-gl hyphen-gu hyphen-hi hyphen-hr hyphen-hu \
        hyphen-id hyphen-is hyphen-it hyphen-kn hyphen-lt hyphen-lv hyphen-ml hyphen-mn hyphen-mr hyphen-nl \
        hyphen-no hyphen-or hyphen-pa hyphen-pl hyphen-pt-br hyphen-pt-pt hyphen-ro hyphen-ru hyphen-sk hyphen-sl \
        hyphen-sr hyphen-sv hyphen-ta hyphen-te hyphen-th hyphen-uk hyphen-zu \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends -t trixie-backports \
        libreoffice-writer libreoffice-calc libreoffice-impress libreoffice-draw python3-uno \
    # unoconverter will look for the Python binary, which has to be at version 3.
    && ln -s /usr/bin/python3 /usr/bin/python \
    # Cleanup.
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# COPY instructions last to maximize cache reuse.

# Copy the Gotenberg binary (LibreOffice variant, no Chromium modules).
COPY --link --from=gotenberg-binary-stage /home/gotenberg-libreoffice /usr/bin/gotenberg

# Copy unoconverter.
COPY --link --from=downloader-stage /downloads/unoconverter /usr/bin/unoconverter

ENV LIBREOFFICE_BIN_PATH=/usr/lib/libreoffice/program/soffice.bin
ENV UNOCONVERTER_BIN_PATH=/usr/bin/unoconverter

USER gotenberg
WORKDIR /home/gotenberg

# Default API port.
EXPOSE 3000

ENTRYPOINT [ "/usr/bin/tini", "--" ]
CMD [ "gotenberg" ]

# ==============================================================
# Cloud Run variants
# ==============================================================

# ----------------------------------------------
# Cloud Run: full (Chromium + LibreOffice)
# ----------------------------------------------
FROM gotenberg AS gotenberg-cloudrun

USER root

# For security reasons, the non-root user gotenberg does not own the Tini binary by default.
# However, some providers like Cloud Run from Google Cloud cannot start a Docker container in that case.
# See https://github.com/gotenberg/gotenberg/issues/90#issuecomment-543551353.
RUN chown gotenberg: /usr/bin/tini

ENV API_PORT_FROM_ENV=PORT
ENV CHROMIUM_AUTO_START=true
ENV LIBREOFFICE_AUTO_START=true
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false
ENV LOG_STD_ENABLE_GCP_FIELDS=true

USER gotenberg

# ----------------------------------------------
# Cloud Run: Chromium only
# ----------------------------------------------
FROM gotenberg-chromium AS gotenberg-cloudrun-chromium

USER root

RUN chown gotenberg: /usr/bin/tini

ENV API_PORT_FROM_ENV=PORT
ENV CHROMIUM_AUTO_START=true
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false
ENV LOG_STD_ENABLE_GCP_FIELDS=true

USER gotenberg

# ----------------------------------------------
# Cloud Run: LibreOffice only
# ----------------------------------------------
FROM gotenberg-libreoffice AS gotenberg-cloudrun-libreoffice

USER root

RUN chown gotenberg: /usr/bin/tini

ENV API_PORT_FROM_ENV=PORT
ENV LIBREOFFICE_AUTO_START=true
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false
ENV LOG_STD_ENABLE_GCP_FIELDS=true

USER gotenberg

# ==============================================================
# AWS Lambda variants
# ==============================================================

# ----------------------------------------------
# AWS Lambda: full (Chromium + LibreOffice)
# ----------------------------------------------
FROM gotenberg AS gotenberg-aws-lambda

USER root

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.9.1 /lambda-adapter /opt/extensions/lambda-adapter

ENV AWS_LWA_PORT=3000
ENV AWS_LWA_READINESS_CHECK_PATH=/health
ENV AWS_LWA_INVOKE_MODE=buffered
ENV API_PORT_FROM_ENV=AWS_LWA_PORT
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false

USER gotenberg

# ----------------------------------------------
# AWS Lambda: Chromium only
# ----------------------------------------------
FROM gotenberg-chromium AS gotenberg-aws-lambda-chromium

USER root

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.9.1 /lambda-adapter /opt/extensions/lambda-adapter

ENV AWS_LWA_PORT=3000
ENV AWS_LWA_READINESS_CHECK_PATH=/health
ENV AWS_LWA_INVOKE_MODE=buffered
ENV API_PORT_FROM_ENV=AWS_LWA_PORT
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false

USER gotenberg

# ----------------------------------------------
# AWS Lambda: LibreOffice only
# ----------------------------------------------
FROM gotenberg-libreoffice AS gotenberg-aws-lambda-libreoffice

USER root

COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.9.1 /lambda-adapter /opt/extensions/lambda-adapter

ENV AWS_LWA_PORT=3000
ENV AWS_LWA_READINESS_CHECK_PATH=/health
ENV AWS_LWA_INVOKE_MODE=buffered
ENV API_PORT_FROM_ENV=AWS_LWA_PORT
ENV WEBHOOK_ENABLE_SYNC_MODE=true
ENV GOTENBERG_BUILD_DEBUG_DATA=false

USER gotenberg
