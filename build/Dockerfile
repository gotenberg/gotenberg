# ARG instructions do not create additional layers. Instead, next layers will
# concatenate them. Also, we have to repeat ARG instructions in each build
# stage that uses them.
ARG GOLANG_VERSION
ARG ALPINE_VERSION

FROM golang:${GOLANG_VERSION:-1.24}-alpine AS go-base
RUN apk update -q && apk add -q --no-cache curl

# ----------------------------------------------
# pdfcpu binary build stage
# ----------------------------------------------
# Note: this stage is required as pdfcpu does not release an armhf variant by
# default.

FROM go-base AS pdfcpu-binary-stage

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
# Font stage
# ----------------------------------------------
FROM alpine:${ALPINE_VERSION:-3.21} AS fonts

WORKDIR /fonts
RUN \
    apk update -q &&\
    apk add -q --no-cache curl tar bzip2 xz &&\
    export DEB_BASE_URL=http://deb.debian.org/debian/pool/main &&\
    # culmus
    curl -L $DEB_BASE_URL/c/culmus/culmus_0.133.orig.tar.gz | tar -x --gzip -f - --wildcards '*.ttf' &&\
    # fonts-beng, fonts-lohit-guru, fonts-lohit-knda, fonts-telu (fonts-lohit-telu)
    curl -L https://releases.pagure.org/lohit/lohit-ttf-20140220.tar.gz | tar -x --gzip -f - --wildcards '*.ttf' &&\
    curl -L https://releases.pagure.org/lohit/lohit-gurmukhi-ttf-2.91.2.tar.gz | tar -x --gzip -f - --wildcards '*.ttf' &&\
    curl -L $DEB_BASE_URL/f/fonts-beng-extra/fonts-beng-extra_3.2.1.orig.tar.gz | tar -x --gzip -f - --wildcards '*.ttf' &&\
    # fonts-hosny-amiri
    curl -OL https://github.com/aliftype/amiri/releases/download/1.001/Amiri-1.001.zip && unzip Amiri-1.001.zip && rm -f Amiri-1.001.zip Amiri-1.001/*html Amiri-1.001/*md Amiri-1.001/*txt &&\
    # fonts-lklug-sinhala
    curl -L $DEB_BASE_URL/f/fonts-lklug-sinhala/fonts-lklug-sinhala_0.6.orig.tar.xz | tar -x --xz -f - --wildcards '*.ttf' &&\
    # fonts-samyak-gujr, fonts-samyak-mlym, fonts-samyak-taml
    mkdir samyak && curl -L $DEB_BASE_URL/f/fonts-samyak/fonts-samyak_1.2.2.orig.tar.gz | tar -x --gzip -f - -C ./samyak --wildcards '*.ttf' &&\
    # fonts-sarai
    mkdir Sarai && curl -L -o Sarai/Sarai_07.ttf https://sourceforge.net/p/indlinux/svn/HEAD/tree/trunk/fonts/sarai/Sarai_07.ttf?format=raw &&\
    # fonts-sil-abyssinica
    curl -LO https://software.sil.org/downloads/r/abyssinica/AbyssinicaSIL-2.300.zip && unzip AbyssinicaSIL-2.300.zip && rm -rf AbyssinicaSIL-2.300.zip AbyssinicaSIL-2.300/*txt AbyssinicaSIL-2.300/documentation AbyssinicaSIL-2.300/fontmanifest.json AbyssinicaSIL-2.300/web &&\
    # fonts-sil-padauk
    curl -LO https://software.sil.org/downloads/r/padauk/Padauk-5.100.zip && unzip Padauk-5.100.zip && rm -rf Padauk-5.100.zip Padauk-5.100/*txt Padauk-5.100/documentation Padauk-5.100/fontmanifest.json Padauk-5.100/web &&\
    # fonts-telu (fonts-telu-extra)
    mkdir telu-extra && curl -L $DEB_BASE_URL/f/fonts-telu-extra/fonts-telu-extra_2.0.orig.tar.gz | tar x --gzip -f - -C telu-extra --wildcards '*.ttf' &&\
    # fonts-telu (fonts-teluguvijayam)
    curl -L $DEB_BASE_URL/f/fonts-teluguvijayam/fonts-teluguvijayam_2.1.orig.tar.xz | tar x --xz -f - --wildcards '*.ttf' &&\
    # fonts-thai-tlwg
    curl -L https://linux.thai.net/pub/thailinux/software/fonts-tlwg/fonts/ttf-tlwg-0.7.3.tar.xz | tar x --xz -f - --wildcards '*.ttf' &&\
    # fonts-arphic-ukai
    curl -L $DEB_BASE_URL/f/fonts-arphic-ukai/fonts-arphic-ukai_0.2.20080216.2.orig.tar.bz2 | tar -x --bzip2 -f - --wildcards '*.ttc' &&\
    # fonts-arphic-uming
    curl -L $DEB_BASE_URL/f/fonts-arphic-uming/fonts-arphic-uming_0.2.20080216.2.orig.tar.bz2 | tar -x --bzip2 -f - --wildcards '*.ttc' &&\
    # fonts-ipafont-mincho, fonts-ipafont-gothic
    curl -L $DEB_BASE_URL/f/fonts-ipafont/fonts-ipafont_00303.orig.tar.gz | tar -x --gzip -f - --wildcards '*.ttf' &&\
    # fonts-unfonts-core
    curl -L $DEB_BASE_URL/f/fonts-unfonts-core/fonts-unfonts-core_1.0.2-080608.orig.tar.xz | tar -x --xz -f - --wildcards '*.ttf' &&\
    # fonts-crosextra-caladea
    curl -OL https://github.com/huertatipografica/Caladea/archive/refs/heads/master.zip && unzip master.zip -x "Caladea-master/sources/*" && mv Caladea-master/fonts/ttf Caladea && rm -rf Caladea/qaCaladea6 Caladea-master master.zip &&\
    # fonts-crosextra-carlito
    curl -OL https://github.com/googlefonts/carlito/archive/refs/heads/main.zip && unzip main.zip && mv carlito-main/fonts/ttf carlito && rm -rf carlito-main main.zip &&\
    # fonts-sil-gentium
    curl -L $DEB_BASE_URL/f/fonts-sil-gentium/fonts-sil-gentium_1.03.orig.tar.xz | tar -x --xz -f - --wildcards '*.ttf' &&\
    # fonts-sil-gentium-basic
    curl -L $DEB_BASE_URL/f/fonts-sil-gentium-basic/fonts-sil-gentium-basic_1.102.orig.tar.xz | tar -x --xz -f - --wildcards '*.ttf'

# ----------------------------------------------
# Final stage
# ----------------------------------------------
FROM alpine:${ALPINE_VERSION:-3.21}

ARG GOTENBERG_VERSION
ARG GOTENBERG_USER_GID
ARG GOTENBERG_USER_UID
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

# Install fonts.
COPY --from=fonts /fonts /home/gotenberg/.fonts
RUN \
    apk update -q &&\
    apk add -q --no-cache \
    font-dejavu \
    font-liberation \
    font-linux-libertine \
    font-noto-all \
    font-noto-cjk \
    font-noto-emoji \
    font-wqy-zenhei \
    msttcorefonts-installer \
    fontconfig \
    &&\
    chown -R gotenberg:gotenberg /home/gotenberg/.fonts &&\
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
    curl -Ls https://raw.githubusercontent.com/gotenberg/unoconverter/v0.1.1/unoconv -o /usr/bin/unoconverter &&\
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