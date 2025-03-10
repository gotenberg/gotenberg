# ARG instructions do not create additional layers. Instead, next layers will
# concatenate them. Also, we have to repeat ARG instructions in each build
# stage that uses them.
ARG GOLANG_VERSION

# ----------------------------------------------
# pdfcpu binary build stage
# ----------------------------------------------
# Note: this stage is required as pdfcpu does not release an armhf variant by
# default.

FROM golang:$GOLANG_VERSION AS pdfcpu-binary-stage

ARG PDFCPU_VERSION
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
FROM golang:$GOLANG_VERSION AS gotenberg-binary-stage

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
# Font stage
# ----------------------------------------------
FROM alpine:3.21 AS fonts

WORKDIR /
RUN \
    apk add bzip2 curl xz &&\
    curl -L https://github.com/google/fonts/archive/master.tar.gz | tar xzf - &&\
    mkdir /culmus && cd /culmus && curl -L http://deb.debian.org/debian/pool/main/c/culmus/culmus_0.133.orig.tar.gz | tar xzf - \
    &&\
    mkdir /samyak && cd /samyak && curl -L http://deb.debian.org/debian/pool/main/f/fonts-samyak/fonts-samyak_1.2.2.orig.tar.gz | tar xzf - \
    &&\
    mkdir /sarai && cd /sarai && curl -L http://deb.debian.org/debian/pool/main/f/fonts-sarai/fonts-sarai_1.0.orig.tar.gz | tar xzf - \
    &&\
    mkdir /arphic-ukai && cd /arphic-ukai && curl -L http://deb.debian.org/debian/pool/main/f/fonts-arphic-ukai/fonts-arphic-ukai_0.2.20080216.2.orig.tar.bz2 | tar xjf - \
    &&\
    mkdir /arphic-uming && cd /arphic-uming && curl -L http://deb.debian.org/debian/pool/main/f/fonts-arphic-uming/fonts-arphic-uming_0.2.20080216.2.orig.tar.bz2 | tar xjf - \
    &&\
    mkdir /unfonts && cd /unfonts && curl -L http://deb.debian.org/debian/pool/main/f/fonts-unfonts-core/fonts-unfonts-core_1.0.2-080608.orig.tar.xz | xz -d | tar xf - \
    &&\
    mkdir /gentium && cd /gentium && curl -L http://deb.debian.org/debian/pool/main/f/fonts-sil-gentium/fonts-sil-gentium_1.03.orig.tar.xz | xz -d | tar xf - \
    &&\
    mkdir /gentium-basic && cd /gentium-basic && curl -L http://deb.debian.org/debian/pool/main/f/fonts-sil-gentium-basic/fonts-sil-gentium-basic_1.102.orig.tar.xz | xz -d | tar xf - 

# ----------------------------------------------
# Final stage
# ----------------------------------------------
FROM alpine:3.21

ARG GOTENBERG_VERSION
ARG GOTENBERG_USER_GID
ARG GOTENBERG_USER_UID
ARG NOTO_COLOR_EMOJI_VERSION
ARG PDFTK_VERSION

LABEL org.opencontainers.image.title="Gotenberg" \
      org.opencontainers.image.description="A containerized API for seamless PDF conversion." \
      org.opencontainers.image.version="$GOTENBERG_VERSION" \
      org.opencontainers.image.authors="Julien Neuhart <neuhart.julien@gmail.com>" \
      org.opencontainers.image.documentation="https://gotenberg.dev" \
      org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

RUN \
    # Create a non-root user.
    # All processes in the Docker container will run with this dedicated user.
    addgroup -g "$GOTENBERG_USER_GID" gotenberg &&\
    adduser -D -u "$GOTENBERG_USER_UID" -G gotenberg -s /bin/bash -h /home/gotenberg gotenberg

RUN \
    # Install system dependencies required for the next instructions or debugging.
    # Note: tini is a helper for reaping zombie processes.
    apk update -q &&\
    apk add -q --no-cache curl gnupg tini python3 openjdk17-jre-headless

COPY --from=fonts /fonts-main/ofl/carlito /home/gotenberg/.fonts/
COPY --from=fonts /fonts-main/ofl/caladea /home/gotenberg/.fonts/
COPY --from=fonts /fonts-main/ofl/amiri /home/gotenberg/.fonts/
COPY --from=fonts /fonts-main/ofl/amiriquran /home/gotenberg/.fonts/
COPY --from=fonts /fonts-main/ofl/abyssinicasil /home/gotenberg/.fonts/
COPY --from=fonts /fonts-main/ofl/padauk /home/gotenberg/.fonts/
COPY --from=fonts /culmus /home/gotenberg/.fonts/
COPY --from=fonts /samyak /home/gotenberg/.fonts/
COPY --from=fonts /sarai /home/gotenberg/.fonts/
COPY --from=fonts /arphic-ukai /home/gotenberg/.fonts/
COPY --from=fonts /arphic-uming /home/gotenberg/.fonts/
COPY --from=fonts /unfonts /home/gotenberg/.fonts/
COPY --from=fonts /gentium /home/gotenberg/.fonts/
COPY --from=fonts /gentium-basic /home/gotenberg/.fonts/

RUN \
    # Install fonts.
    apk update -q &&\
    apk add -q --no-cache \
    font-dejavu \
    font-ipa \
    font-ipaex \
    font-liberation \
    font-linux-libertine \
    font-noto-all \
    font-noto-cjk \
    font-noto-emoji \
    font-wqy-zenhei \
    msttcorefonts-installer \
    fontconfig \
    &&\
    update-ms-fonts &&\
    fc-cache -fv &&\
    fc-list

RUN \
    # Install Chromium.
    apk update -q &&\
    apk add -q --no-cache chromium &&\
    # Verify installation.
    chromium --version

RUN \
    # Install LibreOffice & unoconverter.
    apk update -q &&\
    apk add -q --no-cache libreoffice &&\
    curl -Ls https://raw.githubusercontent.com/gotenberg/unoconverter/v0.0.1/unoconv -o /usr/bin/unoconverter &&\
    chmod +x /usr/bin/unoconverter &&\
    apk add -q --no-cache py3-setuptools py3-looseversion &&\
    # Verify installations.
    libreoffice --version &&\
    unoconverter --version

RUN \
    # Install PDFtk, QPDF & ExifTool (PDF engines).
    # See https://github.com/gotenberg/gotenberg/pull/273.
    curl -o /usr/bin/pdftk-all.jar "https://gitlab.com/api/v4/projects/5024297/packages/generic/pdftk-java/$PDFTK_VERSION/pdftk-all.jar" &&\
    echo -e '#!/bin/sh\n\nexec java -jar /usr/bin/pdftk-all.jar "$@"' > /usr/bin/pdftk && \
    chmod +x /usr/bin/pdftk &&\
    apk update -q &&\
    apk add -q --no-cache qpdf exiftool &&\
    # Verify installations.
    pdftk --version &&\
    qpdf --version &&\
    exiftool --version

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