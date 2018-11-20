FROM golang:1.11-stretch

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | Common libraries
# |--------------------------------------------------------------------------
# |
# | Libraries used in the build process of this image.
# |

RUN echo "deb http://httpredir.debian.org/debian/ stretch main contrib non-free" > /etc/apt/sources.list &&\
    apt-get update &&\
    apt-get install -y curl wget

# |--------------------------------------------------------------------------
# | PM2
# |--------------------------------------------------------------------------
# |
# | Installs PM2 for launching programs in background and with failure 
# | recovering. In our case: Chrome (headless) and Office (headless). 
# |

RUN curl -sL https://raw.githubusercontent.com/Unitech/pm2/master/packager/setup.deb.sh | bash -

# |--------------------------------------------------------------------------
# | Chrome
# |--------------------------------------------------------------------------
# |
# | Installs Chrome for HTML and Markdown conversions.
# |

RUN wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | apt-key add - &&\
    echo "deb http://dl.google.com/linux/chrome/deb/ stable main" | tee /etc/apt/sources.list.d/google-chrome.list &&\
    apt-get update &&\
    apt-get -y install google-chrome-stable

# |--------------------------------------------------------------------------
# | Unoconv
# |--------------------------------------------------------------------------
# |
# | Installs MS fonts and unoconv for Office documents conversions.
# |

RUN apt-get -y install ttf-mscorefonts-installer unoconv

# |--------------------------------------------------------------------------
# | GolangCI-Lint
# |--------------------------------------------------------------------------
# |
# | Installs GolangCI-Lint, a linters Runner for Go. 5x faster 
# | than gometalinter.
# |

RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions to setup CI.
# |

# Defines our workding outside of $GOPATH (we're using go modules).
WORKDIR /ci

# Copies our module dependencies definitions.
COPY go.mod .
COPY go.sum .

# Installs project dependencies.
RUN go get

ENTRYPOINT ["build/ci/docker-entrypoint.sh"]