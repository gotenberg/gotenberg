---
title: Scalability
---

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