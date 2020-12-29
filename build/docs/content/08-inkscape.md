---
title: Inkscape
---

Gotenberg provides the endpoint `/convert/inkscape` for SVG document conversions.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

You may send one file wit the extensions `.svg`.

The file will be converted into a PDF file.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/inkscape \
    --header 'Content-Type: multipart/form-data' \
    --form files=@document.svg \
    -o result.pdf
```
