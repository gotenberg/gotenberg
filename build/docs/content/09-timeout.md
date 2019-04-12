---
title: Timeout
---

All endpoints accept a form value named `waitTimeout`.

The API will wait the given seconds before it considers the conversion to be unsucessful.

It takes a string representation of a float as value (e.g `30` for 30 seconds).

> You may also define this value globally: see the [environment variables](#environment_variables.default_wait_timeout) section.