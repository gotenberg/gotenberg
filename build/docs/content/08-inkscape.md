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
    --form files=@gotenberg.svg \
    -o result.pdf
```

## Image links

You can have links to external image in your svg file and
provides thoses images in the same API call.

The image link should be relative with no extra folders.
Here is a SVG snippet showing the image `ref` :

```xml
<image sodipodi:absref="image1.png" xlink:href="image1.png" y="22.266037" x="34.830791" id="image177" preserveAspectRatio="none" height="154.86531" width="151.07822" />
<image sodipodi:absref="image2.gif" xlink:href="image2.gif" width="75.935417" height="84.666664" preserveAspectRatio="none" id="image255" x="9.2691526" y="195.45641" />
<image sodipodi:absref="image3.jpeg" xlink:href="image3.jpeg" width="70.527336" height="69.172668" preserveAspectRatio="none" id="image325" x="130.69456" y="208.5488" />
```

You can convert the SVG with its external images into PDF in a single call :

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/inkscape \
    --header 'Content-Type: multipart/form-data' \
    --form files=@gotenberg_ter.svg \
    --form files=@image1.png \
    --form files=@image2.gif \
    --form files=@image3.jpeg \
    -o result_images.pdf
```
