---
title: Fonts
---

By default, a handful of fonts are installed. Asian characters are also supported out of the box.

If you wish to use more fonts, you will have to create your own image:

```Dockerfile
FROM thecodingmachine/gotenberg:6

RUN apt-get -y install yourfonts
```
