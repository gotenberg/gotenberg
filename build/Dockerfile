ARG GOLANG_VERSION
ARG GOTENBERG_VERSION

FROM golang:$GOLANG_VERSION AS binary-stage

ARG GOTENBERG_VERSION
ENV CGO_ENABLED 0
WORKDIR /home

COPY go.mod go.sum ./
RUN go mod download &&\
    go mod verify

COPY cmd ./cmd
COPY pkg ./pkg
RUN go build -o gotenberg -ldflags "-X 'github.com/gotenberg/gotenberg/v7/cmd.Version=$GOTENBERG_VERSION'" cmd/gotenberg/main.go

FROM alpine:3.18

ARG GOTENBERG_VERSION
ARG GOTENBERG_USER_GID
ARG GOTENBERG_USER_UID
ARG NOTO_COLOR_EMOJI_VERSION
ARG PDFTK_VERSION

LABEL author="Hayati Ibis" \
    description="A Docker-powered stateless API for PDF files." \
    github="https://github.com/gotenberg/gotenberg" \
    version="$GOTENBERG_VERSION" \
    website="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"

RUN \
    addgroup -g "$GOTENBERG_USER_GID" gotenberg && \
    adduser -u "$GOTENBERG_USER_UID" -G gotenberg -h /home/gotenberg -D gotenberg && \
    mkdir -p /home/gotenberg && \
    chown gotenberg:gotenberg /home/gotenberg

RUN apk --no-cache add curl bash htop tini openjdk8-jre-base bash python3

# Fonts
RUN apk --no-cache add msttcorefonts-installer fontconfig && \
    update-ms-fonts && \
    fc-cache -f

RUN apk add chromium &&\
    apk version chromium

RUN apk add libreoffice &&\
    curl -Ls https://raw.githubusercontent.com/dagwieers/unoconv/master/unoconv -o /usr/local/bin/unoconv &&\
    chmod +x /usr/local/bin/unoconv &&\
    # Verify installations.
    libreoffice --version &&\
    unoconv --version &&\
    # Cleanup.
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN \
    # Install PDFtk & QPDF (PDF engines).
    # See https://github.com/gotenberg/gotenberg/pull/273.
    curl -o /usr/local/bin/pdftk.jar "https://gitlab.com/api/v4/projects/5024297/packages/generic/pdftk-java/$PDFTK_VERSION/pdftk-all.jar" &&\
    chmod 775 /usr/local/bin/pdftk.jar &&\
    echo '#!/usr/bin/env bash\n\n java -jar "$0.jar" "$@"' > /usr/local/bin/pdftk && \
    chmod 775 /usr/local/bin/pdftk &&\
    apk add qpdf &&\
    # Verify installations.
    pdftk --version &&\
    qpdf --version &&\
    # Cleanup.
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Improve fonts subpixel hinting and smoothing.
# Credits:
# https://github.com/arachnys/athenapdf/issues/69.
# https://github.com/arachnys/athenapdf/commit/ba25a8d80a25d08d58865519c4cd8756dc9a336d.
COPY build/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# Copy the Gotenberg binary from the binary stage.
COPY --from=binary-stage /home/gotenberg /usr/bin/

ENV GC_EXCLUDE_SUBSTR "hsperfdata_root,hsperfdata_gotenberg"
ENV CHROMIUM_BIN_PATH /usr/bin/chromium
ENV UNOCONV_BIN_PATH /usr/local/bin/unoconv
ENV LIBREOFFICE_BIN_PATH /usr/lib/libreoffice/program/soffice.bin
ENV PDFTK_BIN_PATH /usr/local/bin/pdftk
ENV QPDF_BIN_PATH /usr/bin/qpdf

USER gotenberg
WORKDIR /home/gotenberg

EXPOSE 3000

ENTRYPOINT [ "/sbin/tini", "--" ]
CMD [ "gotenberg" ]