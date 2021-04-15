FROM debian:buster-slim

# |--------------------------------------------------------------------------
# | Common libraries
# |--------------------------------------------------------------------------
# |
# | Libraries used in the build process of this image.
# |
RUN apt-get update &&\
    apt-get --no-install-recommends install -y ca-certificates curl gnupg procps &&\
    rm -rf /var/lib/apt/lists/*

# |--------------------------------------------------------------------------
# | Microsoft font installer
# |--------------------------------------------------------------------------
RUN apt-get update &&\
    curl -o ./ttf-mscorefonts-installer_3.8_all.deb http://httpredir.debian.org/debian/pool/contrib/m/msttcorefonts/ttf-mscorefonts-installer_3.8_all.deb &&\
    apt --no-install-recommends install -y ./ttf-mscorefonts-installer_3.8_all.deb && rm ./ttf-mscorefonts-installer_3.8_all.deb &&\
    rm -rf /var/lib/apt/lists/*

# |--------------------------------------------------------------------------
# | Chrome
# |--------------------------------------------------------------------------
# |
# | Installs Chrome.
# |
RUN curl https://dl.google.com/linux/linux_signing_key.pub | apt-key add - &&\
    echo "deb http://dl.google.com/linux/chrome/deb/ stable main" | tee /etc/apt/sources.list.d/google-chrome.list &&\
    apt-get update &&\
    apt-get install --no-install-recommends -y --allow-unauthenticated google-chrome-stable &&\
    rm -rf /var/lib/apt/lists/*

# |--------------------------------------------------------------------------
# | LibreOffice
# |--------------------------------------------------------------------------
# |
# | Installs LibreOffice.
# |

# https://github.com/nextcloud/docker/issues/380
RUN mkdir -p /usr/share/man/man1mkdir -p /usr/share/man/man1 &&\ 
    echo "deb http://httpredir.debian.org/debian/ buster-backports main contrib non-free" >> /etc/apt/sources.list &&\
    apt-get update &&\
    apt-get --no-install-recommends -t buster-backports -y install libreoffice &&\
    rm -rf /var/lib/apt/lists/*

# |--------------------------------------------------------------------------
# | Unoconv
# |--------------------------------------------------------------------------
# |
# | Installs unoconv.
# |

ENV UNO_URL=https://raw.githubusercontent.com/dagwieers/unoconv/master/unoconv

RUN curl -Ls $UNO_URL -o /usr/bin/unoconv &&\
    chmod +x /usr/bin/unoconv &&\
    ln -s /usr/bin/python3 /usr/bin/python &&\
    unoconv --version

# |--------------------------------------------------------------------------
# | PDFtk
# |--------------------------------------------------------------------------
# |
# | Installs PDFtk as an alternative to pdfcpu for merging PDFs.
# | https://github.com/thecodingmachine/gotenberg/issues/29
# |

ARG PDFTK_VERSION=924565150

RUN curl -o /usr/bin/pdftk "https://gitlab.com/pdftk-java/pdftk/-/jobs/${PDFTK_VERSION}/artifacts/raw/build/native-image/pdftk" \
    && chmod a+x /usr/bin/pdftk

# |--------------------------------------------------------------------------
# | Fonts
# |--------------------------------------------------------------------------
# |
# | Installs a handful of fonts.
# | Note: ttf-mscorefonts-installer are installed on top of this Dockerfile.
# |

# Credits: 
# https://github.com/arachnys/athenapdf/blob/master/cli/Dockerfile
# https://help.accusoft.com/PrizmDoc/v12.1/HTML/Installing_Asian_Fonts_on_Ubuntu_and_Debian.html
RUN apt-get update &&\
    apt-get install --no-install-recommends -y \
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
    rm -rf /var/lib/apt/lists/*

COPY build/base/fonts/* /usr/local/share/fonts/
COPY build/base/fonts.conf /etc/fonts/conf.d/100-gotenberg.conf

# |--------------------------------------------------------------------------
# | Default user
# |--------------------------------------------------------------------------
# |
# | All processes in the Docker container will run as a dedicated 
# | non-root user.
# |

ARG GOTENBERG_USER_GID=1001
ARG GOTENBERG_USER_UID=1001

RUN groupadd --gid ${GOTENBERG_USER_GID} gotenberg \
  && useradd --uid ${GOTENBERG_USER_UID} --gid gotenberg --shell /bin/bash --home /gotenberg --no-create-home gotenberg \
  && mkdir /gotenberg \
  && chown gotenberg: /gotenberg