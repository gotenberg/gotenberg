---
title: Fonts
---

By default, a handful of fonts are installed. Asian characters are also supported out of the box.

If you wish to use more fonts, you will have to create your own image:

```Dockerfile
FROM thecodingmachine/gotenberg:6

USER root

# add fonts from repo
RUN apt-get -y install yourfonts

# add fonts directly
COPY my/fonts/* /usr/local/share/fonts/

USER gotenberg
```
