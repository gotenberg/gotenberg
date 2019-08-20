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
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$dirPath = "/foo";
$filename = $client->store($request, $dirPath);
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

> **Attention:** the CSS properties are independant of the ones used in the `index.html` file.
> Also, `footer.html` CSS properties override the ones from `header.html`.

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
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.Header("header.html")
    req.Footer("footer.html")
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$header = DocumentFactory::makeFromPath('header.html', 'header.html');
$footer = DocumentFactory::makeFromPath('footer.html', 'footer.html');
$request = new HTMLRequest($index);
$request->setHeader($header);
$request->setFooter($footer);
$dirPath = "/foo";
$filename = $client->store($request, $dirPath);
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
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.Assets("font.woff", "img.gif", "style.css")
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$assets = [
    DocumentFactory::makeFromPath('style.css', 'style.css'),
    DocumentFactory::makeFromPath('img.png', 'img.png'),
    DocumentFactory::makeFromPath('font.woff', 'font.woff'),
];
$request = new HTMLRequest($index);
$request->setAssets($assets);
$dest = "result.pdf";
$client->store($request, $dest);
```

## Paper size, margins, orientation

You may also customize the resulting PDF format.

By default, it will be rendered with `A4` size, `1 inch` margins and `portrait` orientation.

> Paper size and margins have to be provided in `inches`. Same for margins.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form paperWidth=8.27 \
    --form paperHeight=11.27 \
    --form marginTop=0 \
    --form marginBottom=0 \
    --form marginLeft=0 \
    --form marginRight=0 \
    --form landscape=true \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.PaperSize(gotenberg.A4)
    req.Margins(gotenberg.NoMargins)
    req.Landscape(true)
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$request->setPaperSize(Request::A4);
$request->setMargins(Request::NO_MARGINS);
$request->setLandscape(true);
$dest = "result.pdf";
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
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.WaitDelay(5.5)
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$request->setWaitDelay(5.5);
$dest = "result.pdf";
$client->store($request, $dest);
```
