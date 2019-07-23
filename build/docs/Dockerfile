FROM thecodingmachine/gotenberg:workspace

# |--------------------------------------------------------------------------
# | static
# |--------------------------------------------------------------------------
# |
# | Installs the static site anti-framework – general-purpose library, 
# | purpose-built commands for various domains.
# |

RUN go get github.com/apex/static/cmd/static-docs

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions of this build.
# |

WORKDIR /gotenberg/docs

CMD [ "static-docs", "--in", "build/docs/content", "--out", "docs", "--theme", "gotenberg", "--title", "Gotenberg", "--subtitle", "A Docker-powered stateless API for converting HTML, Markdown and Office documents to PDF." ]