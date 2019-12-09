---
title: Install
---

Gotenberg is shipped within a Docker image.

You may start it with:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:6
```

> The API will be available at [http://localhost:3000](http://localhost:3000).

The image uses a dedicated non-root user called `gotenberg` with uid and gid `1001`.

If you wish to change those uid and gid, you will have to:

* clone the project
* re-build the image
* publish the image in your own Docker registry

For instance:

```bash
$ git clone https://github.com/thecodingmachine/gotenberg.git
$ make publish GOTENBERG_USER_GID=your_custom_gid GOTENBERG_USER_UID=your_custom_uid DOCKER_REGISTRY=your_registry DOCKER_USER=registry_user DOCKER_PASSWORD=registry_password VERSION=6.1.0 
```

> `master` branch is always up-to-date with the latest version of the API.

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

In the deployment specification of the pod, also specify the uid of the user `gotenberg`:

```
securityContext:
  privileged: false
  runAsUser: 1001
```

## Cloud Run (Google Cloud)

If you're looking for cost savings, you might be interested by [Cloud Run](https://cloud.google.com/run).

In the following examples, we will assume your
Gotenberg API is available at [http://localhost:3000](http://localhost:3000).
