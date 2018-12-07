FROM thecodingmachine/gotenberg:base

LABEL authors="Julien Neuhart <j.neuhart@thecodingmachine.com>"

# |--------------------------------------------------------------------------
# | Final touch
# |--------------------------------------------------------------------------
# |
# | Last instructions to setup Gotenberg.
# |

COPY build/package/gotenberg /usr/local/bin/

WORKDIR /gotenberg

EXPOSE 3000
CMD [ "gotenberg" ]