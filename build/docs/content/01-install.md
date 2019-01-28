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

In the following examples, we will assume your
Gotenberg API is available at [http://localhost:3000](http://localhost:3000).

## Go client

```bash
$ go get -u github.com/thecodingmachine/gotenberg
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