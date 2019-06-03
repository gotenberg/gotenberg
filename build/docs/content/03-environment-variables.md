---
title: Environment variables
---

You may customize the API behaviour thanks to environment variables.

## Disable Google Chrome

In order to save some resources, the Gotenberg image accepts the environment variable `DISABLE_GOOGLE_CHROME`.

It takes the strings `"0"` or `"1"` as value.

> If Google Chrome is disabled, the following conversions will **not** be available anymore:
> [HTML](#html), [URL](#url) and [Markdown](#markdown)

## Disable LibreOffice (unoconv)

You may also disable LibreOffice (unoconv) with `DISABLE_UNOCONV`.

> If LibreOffice (unoconv) is disabled, the following conversion will **not** be available anymore:
> [Office](#office)

## Default wait timeout

By default, the API will wait 10 seconds before it considers the conversion to be unsuccessful.

You may customize this timeout thanks to the environment variable `DEFAULT_WAIT_TIMEOUT`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).

> The default timeout may also be overridden per request thanks to the form field `waitTimeout`.
> See the [timeout section](#timeout).

## Disable logging on healthcheck

By default, the API will add a log entry when the [healthcheck endpoint](#ping) is called.

You may turn off this logging so as to avoid unnecessary entries in your logs with the environment variable `DISABLE_HEALTHCHECK_LOGGING`.

This environment variable operates in the same manner as the `DISABLE_GOOGLE_CHROME` and `DISABLE_UNOCONV` variables operate in that it accepts the strings `"0"` or `"1"` as values. 

## Default listen port

By default, the API will listen on port `3000`. For most use cases this is perfectly fine, but at times there may be cases where you need to change this due to port conflicts.

You may customize this port location with the environment variable `DEFAULT_LISTEN_PORT`.

This environment variable accepts any string that can be turned into a port number (e.g., the string `"0"` up to the string `"65535"`).