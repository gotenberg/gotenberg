FROM debian:stretch-slim

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | Libraries
# |--------------------------------------------------------------------------
# |
# | Installs all required libraries.
# |

RUN echo "deb http://httpredir.debian.org/debian/ stretch main contrib non-free" > /etc/apt/sources.list &&\
    apt-get update &&\
    apt-get install -y xvfb wkhtmltopdf pdftk unoconv ttf-mscorefonts-installer &&\
    ln -s /usr/bin/xvfb-run /usr/local/bin/xvfb-run &&\
    ln -s /usr/bin/wkhtmltopdf /usr/local/bin/wkhtmltopdf &&\
    ln -s /usr/bin/pdftk /usr/local/bin/pdftk &&\
    ln -s /usr/bin/unoconv /usr/local/bin/unoconv

# |--------------------------------------------------------------------------
# | Gotenberg
# |--------------------------------------------------------------------------
# |
# | All Gotenberg related stuff.
# |

COPY .build/gotenberg /usr/bin/gotenberg
RUN ln -s /usr/bin/gotenberg /usr/local/bin/gotenberg

WORKDIR /gotenberg
COPY .build/gotenberg.yml /gotenberg/gotenberg.yml

EXPOSE 3000
CMD ["gotenberg"]