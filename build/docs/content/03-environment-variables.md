---
title: Environment variables
---

You may customize the API behaviour thanks to environment variables.

## Log level

The API provides structured logging allowing you to have relevant information
about what's going on.

> If a TTY is attached, the log entries are displayed in text format with colors, otherwise in JSON format.

You may customize the severity of the log entries thanks to the environment variable `LOG_LEVEL`.

It accepts one of the following severities: `"DEBUG"`, `"INFO"` (default) and `"ERROR"`.

## Default listen port

By default, the API will listen on port `3000`.

You may customize this value with the environment variable `DEFAULT_LISTEN_PORT`.

This environment variable accepts any string that can be turned into a port number.

## Root path

By default, the API root path is `/`.

You may customize this value with the environment variable `ROOT_PATH`.

This environment variable accepts a string starting and ending with `/`.

For instance, `/gotenberg/` is a valid value while `gotenberg` is not.

> This is useful if you wish to do service discovery via URL paths.

## Disable Google Chrome

In order to save some resources, the Gotenberg image accepts the environment variable `DISABLE_GOOGLE_CHROME`
for disabling Google Chrome.

It takes the strings `"0"` or `"1"` as value where `1` means `true`

> If Google Chrome is disabled, the following conversions will **not** be available anymore:
> [HTML](#html), [URL](#url) and [Markdown](#markdown)

## Default Google Chrome rpcc buffer size

When performing a [HTML](#html), [URL](#url) or [Markdown](#markdown) conversion, the API might return
a `400` HTTP code with the message `increase the Google Chrome rpcc buffer size`.

If so, you may increase this buffer size with the environment variable `DEFAULT_GOOGLE_CHROME_RPCC_BUFFER_SIZE`.

It takes a string representation of an int as value (e.g. `"1048576"` for 1 MB).
The hard limit is 100 MB and is defined by Google Chrome itself.

> The default Google Chrome rpcc buffer size may also be overridden per request thanks to the form field `googleChromeRpccBufferSize`.
> See the [rpcc buffer size section](#html.rpcc_buffer_size).

## Google Chrome ignore certificate errors

When performing a [URL](#url) conversion, Google Chrome will not accept certificate errors. 

You may allow insecure connections by setting the `GOOGLE_CHROME_IGNORE_CERTIFICATE_ERRORS` environment variable to `"1"`.

**You should be careful with this feature and only enable it in your development environment.**

## Disable LibreOffice (unoconv)

You may also disable LibreOffice (unoconv) with `DISABLE_UNOCONV`.

> If LibreOffice (unoconv) is disabled, the following conversion will **not** be available anymore:
> [Office](#office)

## Default wait timeout

By default, the API will wait 10 seconds before it considers the conversion to be unsuccessful.
If unsucessful, it returns a `504` HTTP code.

You may customize this timeout thanks to the environment variable `DEFAULT_WAIT_TIMEOUT`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).

> The default timeout may also be overridden per request thanks to the form field `waitTimeout`.
> See the [timeout section](#timeout).

## Maximum wait timeout

By default, the value of the form field `waitTimeout` cannot be more than 30 seconds.

You may increase or decrease this limit thanks to the environment variable `MAXIMUM_WAIT_TIMEOUT`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).

## Default webhook URL timeout

By default, the API will wait 10 seconds before it considers the sending of the resulting PDF to be unsuccessful.

> See the [webhook section](#webhook).

You may customize this timeout thanks to the environment variable `DEFAULT_WEBHOOK_URL_TIMEOUT`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).

> The default timeout may also be overridden per request thanks to the form field `webhookURLTimeout`.
> See the [webhook timeout section](#webhook.timeout).

## Maximum webhook URL timeout

By default, the value of the form field `webhookURLTimeout` cannot be more than 30 seconds.

You may increase or decrease this limit thanks to the environment variable `MAXIMUM_WEBHOOK_URL_TIMEOUT`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).

## Maximum wait delay

By default, the value of the form field `waitDelay` cannot be more than 10 seconds.

> See the [wait delay section](#html.wait_delay).

You may increase or decrease this limit thanks to the environment variable `MAXIMUM_WAIT_DELAY`.

It takes a string representation of a float as value (e.g `"2.5"` for 2.5 seconds).
