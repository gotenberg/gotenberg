---
title: URL
---

Gotenberg provides the endpoint `/convert/url` for remote URL conversions.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

This endpoint does not accept an `index.html` file nor assets files but a form field
named `remoteURL` instead. Otherwise, URL conversions work the same as HTML conversions.

> **Attention:** when converting a website to PDF, you should remove all margins.
> If not, some of the content of the page might be hidden.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/url \
    --header 'Content-Type: multipart/form-data' \
    --form remoteURL=https://google.com \
    --form marginTop=0 \
    --form marginBottom=0 \
    --form marginLeft=0 \
    --form marginRight=0 \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
req := gotenberg.NewURLRequest("https://google.com")
req.Margins(gotenberg.NoMargins)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\URLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$request = new URLRequest('https://google.com');
$request->setMargins(Request::NO_MARGINS);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Custom HTTP headers

You may send your own HTTP headers to the `remoteURL`.

For instance, by adding the HTTP header `Gotenberg-Remoteurl-Your-Header` to your request,
the API will send a request to the `remoteURL` with the HTTP header `Your-Header`.

> **Attention:** the API uses a canonical format for the HTTP headers:
> it transforms the first
> letter and any letter following a hyphen to upper case;
> the rest are converted to lowercase. For example, the
> canonical key for `accept-encoding` is `Accept-Encoding`.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/url \
    --header 'Content-Type: multipart/form-data' \
    --header 'Gotenberg-Remoteurl-Your-Header: Foo' \
    --form remoteURL=https://google.com \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
req := gotenberg.NewURLRequest("https://google.com")
req.AddRemoteURLHTTPHeader("Your-Header", "Foo")
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\URLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$request = new URLRequest('https://google.com');
$request->addRemoteURLHTTPHeader('Your-Header', 'Foo')
$dest = 'result.pdf';
$client->store($request, $dest);
```
