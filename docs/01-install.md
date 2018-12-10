---
title: Install
---

Gotenberg is shipped within a Docker image.

You may start it with:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:3
```

Or add it in your Docker Compose stack:

```yaml
version: '3'

services:

  # your others services

  gotenberg:
    image: thecodingmachine/gotenberg:3
```

It may also be deployed with Kubernetes.