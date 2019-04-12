---
title: Install
---

Gotenberg is shipped within a Docker image.

You may start it with:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:5
```

> The API will be available at [http://localhost:3000](http://localhost:3000).

## Docker Compose

You may also add it in your Docker Compose stack:

```yaml
version: '3'

services:

  # your others services

  gotenberg:
    image: thecodingmachine/gotenberg:5
```

> The API will be available under `gotenberg:3000` in your Docker Compose network.

## Kubernetes

It may also be deployed with Kubernetes.

Make sure to provide enough memory and CPU requests (for instance `512Mi` and `0.2` CPU).
Otherwise the API will not be able to launch Google Chrome and LibreOffice (unoconv).

> The more resources are granted, the quicker will be the conversions.

In the following examples, we will assume your
Gotenberg API is available at [http://localhost:3000](http://localhost:3000).