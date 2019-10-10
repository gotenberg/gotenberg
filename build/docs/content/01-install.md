---
title: Install
---

Gotenberg is shipped within a Docker image.

> It uses a dedicated non-root user called `gotenberg` with uid and gid `1001`.

You may start it with:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:6
```

> The API will be available at [http://localhost:3000](http://localhost:3000).

## Docker Compose

You may also add it in your Docker Compose stack:

```yaml
version: '3'

services:

  # your others services

  gotenberg:
    image: thecodingmachine/gotenberg:6
```

> The API will be available under `gotenberg:3000` in your Docker Compose network.

## Kubernetes

It may also be deployed with Kubernetes.

Make sure to provide enough memory and CPU requests (for instance `512Mi` and `0.2` CPU).

> The more resources are granted, the quicker will be the conversions.

In the deployment specification of the pod, also specify the uid `1001` of the user `gotenberg`:

```
securityContext:
  privileged: false
  runAsUser: 1001
```

In the following examples, we will assume your
Gotenberg API is available at [http://localhost:3000](http://localhost:3000).
