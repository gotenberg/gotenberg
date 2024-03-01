# ARG instructions do not create additional layers. Instead, next layers will
# concatenate them. Also, we have to repeat ARG instructions in each build
# stage that uses them.
ARG GOLANG_VERSION

# ----------------------------------------------
# Gotenberg binary build stage
# ----------------------------------------------
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

# ----------------------------------------------
# Final stage
# ----------------------------------------------
FROM alpine:3

ARG GOTENBERG_VERSION
ARG GOTENBERG_USER_GID
ARG GOTENBERG_USER_UID
ARG NOTO_COLOR_EMOJI_VERSION
ARG PDFTK_VERSION
ARG TMP_CHOMIUM_VERSION_ARMHF="116.0.5845.180-1~deb12u1"

LABEL author="Hayati Ibis" \
    description="Alpine Based A Docker-powered stateless API for PDF files." \
    github="https://github.com/hayatibis/gotenberg" \
    version="$GOTENBERG_VERSION" \
    website="https://gotenberg.dev" \
    org.opencontainers.image.source="https://github.com/gotenberg/gotenberg"


RUN \
    # Create a non-root user.
    # All processes in the Docker container will run with this dedicated user.
    addgroup -g "$GOTENBERG_USER_GID" gotenberg && \
    adduser -u "$GOTENBERG_USER_UID" -G gotenberg -h /home/gotenberg -D gotenberg && \
    mkdir -p /home/gotenberg && \
    chown gotenberg:gotenberg /home/gotenberg


RUN \
    # Install system dependencies required for the next instructions or debugging.
    # Note: tini is a helper for reaping zombie processes.
    apk update && apk --no-cache add curl bash htop tini openjdk8-jre-base bash python3 &&\
    rm -rf /var/cache/apk/*


RUN \
    # Install fonts.
    apk --no-cache add \
    font-noto-all \
    font-carlito \
    font-dejavu \
    font-dejavu-sans-mono-nerd \
    font-liberation \
    ttf-liberation \
    font-liberation-sans-narrow \
    font-liberation-mono-nerd \
    font-linux-libertine \
    font-noto-all \
    font-croscore \
    font-terminus \
    font-bakoma \
    font-awesome \
    font-inconsolata \
    font-mononoki \
    font-freefont \
    msttcorefonts-installer fontconfig &&\
    update-ms-fonts && \
    fc-cache -f &&\
    rm -rf /var/cache/apk/*

RUN apk --no-cache add \
    font-wqy-zenhei --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/

RUN apk --no-cache add chromium &&\
    apk version chromium &&\
    rm -rf /var/cache/apk/*

RUN apk --no-cache add libreoffice &&\
    curl -Ls https://raw.githubusercontent.com/dagwieers/unoconv/master/unoconv -o /usr/local/bin/unoconv &&\
    chmod +x /usr/local/bin/unoconv &&\
    # Verify installations.
    libreoffice --version &&\
    unoconv --version &&\
    # Cleanup.
    rm -rf /var/cache/apk/*

RUN \
    # Install PDFtk & QPDF (PDF engines).
    # See https://github.com/gotenberg/gotenberg/pull/273.
    curl -o /usr/local/bin/pdftk.jar "https://gitlab.com/api/v4/projects/5024297/packages/generic/pdftk-java/$PDFTK_VERSION/pdftk-all.jar" &&\
    chmod 775 /usr/local/bin/pdftk.jar &&\
    echo '#!/usr/bin/bash\n\n java -jar "$0.jar" "$@"' > /usr/local/bin/pdftk && \
    chmod 775 /usr/local/bin/pdftk &&\
    apk --no-cache add qpdf &&\
    # Verify installations.
    # pdftk --version &&\
    # qpdf --version &&\
    # Cleanup.
    rm -rf /var/cache/apk/*

# Improve fonts subpixel hinting and smoothing.
# Credits:
# https://github.com/arachnys/athenapdf/issues/69.
# https://github.com/arachnys/athenapdf/commit/ba25a8d80a25d08d58865519c4cd8756dc9a336d.
COPY build/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# Copy the Gotenberg binary from the binary stage.
COPY --from=binary-stage /home/gotenberg /usr/bin/

# Environment variables required by modules or else.
ENV CHROMIUM_BIN_PATH /usr/bin/chromium
ENV LIBREOFFICE_BIN_PATH /usr/lib/libreoffice/program/soffice.bin
ENV UNOCONVERTER_BIN_PATH /usr/bin/unoconverter
ENV PDFTK_BIN_PATH /usr/bin/pdftk
ENV QPDF_BIN_PATH /usr/bin/qpdf

USER gotenberg
WORKDIR /home/gotenberg

EXPOSE 3000

ENTRYPOINT [ "/sbin/tini", "--" ]
CMD [ "gotenberg" ]