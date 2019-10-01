---
title: Scalability
---

Google Chrome and unoconv (LibreOffice) are intricate programs.

Gotenberg tries to abstract as much complexity as possible but it can
only do it to a certain extend.

For instance, Google Chrome misbehaves if there are too many concurrent conversions.
That's why for [HTML](#html), [URL](#url) and [Markdown](#markdown) conversions, the API does only 6 conversions in parallel.
The more concurrent requests, the more `504` HTTP codes the API will return.

On another hand, for [Office](#office) conversions, the API will start as many unoconv (LibreOffice) instances as there are
requests. The limitation here is the available memory.

> See our [load testing use case](../loadtesting) for more details about the API behaviour under heavy load.

## Strategies

### Increase timeout

This strategy is mostly for [HTML](#html), [URL](#url) and [Markdown](#markdown) conversions.

You may increase the conversion timeout. In other words, you accept that a conversion takes more time
if there are more than 6 conversions in parallel.

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
