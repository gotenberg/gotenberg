---
title: Scalability
---

The API uses under the hood intricate programs.

Gotenberg tries to abstract as much complexity as possible but it can
only do it to a certain extend.

For instance, [Office](#office) and [Merge](#merge) endpoints will start respectively as many LibreOffice (unoconv) and PDTk
instances are there are requests. The limitation here is the available memory and CPU usage.

On another hand, for the [HTML](#html), [URL](#url) and [Markdown](#markdown) endpoints, the API does only 6 conversions in parallel.
Indeed, Google Chrome misbehaves if there are too many concurrent conversions.

**The more concurrent requests, the more `504` HTTP codes the API will return.**

> See our [load testing use case](https://github.com/thecodingmachine/gotenberg/tree/master/loadtesting) for more details about the API behaviour under heavy load.

## Strategies

### Increase timeout

You may increase the conversion timeout. In other words, you accept that a conversion takes more time
if the API is under heavy load.

> See [timeout section](#timeout).

### Scaling

The API being stateless, you may scale it as much as you want.

For instance, using the following Docker Compose file:

```yaml
version: '3'

services:

  # your others services

  gotenberg:
    image: thecodingmachine/gotenberg:6
```

You may now launch your services using:

```bash
$ docker-compose up --scale gotenberg=your_number_of_instances
```

When requesting the Gotenberg service with your client(s), Docker will automatically
redirect a request to a Gotenberg container according to the round-robin strategy.
