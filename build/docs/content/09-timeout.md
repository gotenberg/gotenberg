---
title: Timeout
---

All endpoints accept a form field named `waitTimeout`.

The API will wait the given **seconds** before it considers the conversion to be unsucessful.

It takes a float as value (e.g `2.5` for 2.5 seconds).

> You may also define this value globally: see the [environment variables](#environment_variables.default_wait_timeout) section.