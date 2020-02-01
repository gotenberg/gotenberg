---
title: HTML
---

Gotenberg provides the endpoint `/convert/html` for HTML conversions.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

The only requirement is to send a file named `index.html`: it is the file
which will be converted to PDF.

For instance:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>My PDF</title>
  </head>
  <body>
    <h1>Hello world!</h1>
  </body>
</html>
```

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$request = new HTMLRequest($index);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Header and footer

You may also add a header and/or a footer in the resulting PDF.
Respectively, a file named `header.html` and `footer.html`.

Each of them **has to be a complete HTML document**:

```html
<html>
    <head>
        <style>
            body {
                font-size: 8rem;
                margin: 4rem auto;
            }
        </style>
    </head>
    <body>
        <p>
            <span class="pageNumber"></span> of <span class="totalPages"></span>
        </p>
    </body>
</html>
```

The following classes allow you to inject printing values:

* `date`: formatted print date
* `title`: document title
* `pageNumber`: current page number
* `totalPage`: total pages in the document

There are some limitations:

* JavaScript is not executed
* external resources are not loaded
* the CSS properties are independant of the ones used in the `index.html` file
* `footer.html` CSS properties override the ones from `header.html`
* only fonts installed in the Docker image are loaded (see the [fonts section](#fonts))
* images only work using a `base64` encoded source (`<img src="data:image/png;base64, iVBORw0K... />`)
* `background-color` and `color` CSS properties require an additional `-webkit-print-color-adjust: exact` CSS property in order to work

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form files=@header.html \
    --form files=@footer.html \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
header, _ := gotenberg.NewDocumentFromPath("header.html", "/path/to/file")
footer, _ := gotenberg.NewDocumentFromPath("footer.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.Header(header)
req.Footer(footer)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$header = DocumentFactory::makeFromPath('header.html', '/path/to/file');
$footer = DocumentFactory::makeFromPath('footer.html', '/path/to/file');
$request = new HTMLRequest($index);
$request->setHeader($header);
$request->setFooter($footer);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Assets

You may also send additional files. For instance: images, fonts, stylesheets and so on.

The only requirement is to make sure that their paths
are on the same level as the `index.html` file.

In others words, this will work:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>My PDF</title>
  </head>
  <body>
    <h1>Hello world!</h1>
    <img src="img.png">
  </body>
</html>
```

But this won't:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>My PDF</title>
  </head>
  <body>
    <h1>Hello world!</h1>
    <img src="/foo/img.png">
  </body>
</html>
```

You may also use *remote* paths for Google fonts, images and so on.

> If you want to install fonts directly in the Gotenberg Docker image,
> see to the [fonts section](#fonts).

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form files=@style.css \
    --form files=@img.png \
    --form files=@font.woff \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
style, _ := gotenberg.NewDocumentFromPath("style.css", "/path/to/file")
img, _ := gotenberg.NewDocumentFromPath("img.png", "/path/to/file")
font, _ := gotenberg.NewDocumentFromPath("font.woff", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.Assets(style, img, font)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$assets = [
    DocumentFactory::makeFromPath('style.css', '/path/to/file'),
    DocumentFactory::makeFromPath('img.png', '/path/to/file'),
    DocumentFactory::makeFromPath('font.woff', '/path/to/file'),
];
$request = new HTMLRequest($index);
$request->setAssets($assets);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Paper size, margins, orientation, scaling

You may also customize the resulting PDF format.

By default, it will be rendered with `A4` size, `1 inch` margins and `portrait` orientation and 100% (`1.0`) page scale.

> Paper size and margins have to be provided in `inches`. Same for margins.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form paperWidth=8.27 \
    --form paperHeight=11.69 \
    --form marginTop=0 \
    --form marginBottom=0 \
    --form marginLeft=0 \
    --form marginRight=0 \
    --form landscape=true \
    --form scale=0.75 \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.PaperSize(gotenberg.A4)
req.Margins(gotenberg.NoMargins)
req.Landscape(true)
req.Scale(0.75)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$request = new HTMLRequest($index);
$request->setPaperSize(Request::A4);
$request->setMargins(Request::NO_MARGINS);
$request->setLandscape(true);
$request->setScale(0.75);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Page ranges

You may specify the page ranges to convert.

The format is the same as the one from the print options
of Google Chrome, e.g. `1-5,8,11-13`.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form pageRanges='1-3,5' \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.PageRanges("1-3,5")
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$request = new HTMLRequest($index);
$request->setPageRanges('1-3,5');
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Wait delay

In some cases, you may want to wait a certain amount of time to make sure the
page you're trying to generate is fully rendered. For instance, if your page relies
a lot on JavaScript for rendering.

> The wait delay is a duration in **seconds** (e.g `2.5` for 2.5 seconds).

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form waitDelay=5.5 \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.WaitDelay(5.5)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$request = new HTMLRequest($index);
$request->setWaitDelay(5.5);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Rpcc buffer size

The API might return a `400` HTTP code with the message `increase the Google Chrome rpcc buffer size`.

If so, you may increase this buffer size with a form field named `googleChromeRpccBufferSize`.

It takes an int as value (e.g. `1048576` for 1 MB).
The hard limit is 100 MB and is defined by Google Chrome itself.

> You may also define this value globally: see the [environment variables](#environment_variables.default_google_chrome_rpcc_buffer_size) section.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form googleChromeRpccBufferSize=1048576 \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.GoogleChromeRpccBufferSize(1048576)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$request = new HTMLRequest($index);
$request->setGoogleChromeRpccBufferSize(1048576);
$dest = 'result.pdf';
$client->store($request, $dest);
```
