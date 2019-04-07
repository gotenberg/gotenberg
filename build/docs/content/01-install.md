---
title: Install
---

Gotenberg is shipped within a Docker image.

You may start it with:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:4
```

> The API will be available at [http://localhost:3000](http://localhost:3000).

Or add it in your Docker Compose stack:

```yaml
version: '3'

services:

  # your others services

  gotenberg:
    image: thecodingmachine/gotenberg:4
```

> The API will be available under `gotenberg:3000` in your Docker Compose network.

It may also be deployed with Kubernetes.

> In Kubernetes, make sure to provide enough memory and CPU requests (for instance `512Mi` and `0.2` CPU).
> Otherwise the API will not be able to launch Google Chrome and Libreoffice (unoconv).
>
> Also note the more resources are granted, the quicker will be the conversions.

In the following examples, we will assume your
Gotenberg API is available at [http://localhost:3000](http://localhost:3000).

## Disabling Google Chrome and/or Libreoffice (unoconv)

In order to save some resources, the Gotenberg image accepts two environment variables: `DISABLE_GOOGLE_CHROME` and `DISABLE_UNOCONV`.

Both accept the string `"1"` as value.

> If Google Chrome is disabled, the following conversions will **not** be available anymore:
> [HTML](#html), [URL](#url) and [Markdown](#markdown)
>
> If Libreoffice (unoconv) is disabled, the following conversion will **not** be available anymore:
> [Office](#office)

## Go client

```bash
$ go get -u github.com/thecodingmachine/gotenberg-go-client/v4
```

## PHP client

Unless your project already has a PSR7 `HttpClient`, install `php-http/guzzle6-adapter`:

```bash
$ composer require php-http/guzzle6-adapter
```

Then the PHP client:

```bash
$ composer require thecodingmachine/gotenberg-php-client
```