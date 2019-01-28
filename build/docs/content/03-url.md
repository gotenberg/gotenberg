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
    --form remoteURL=https://google.com
    --form marginTop=0 \
    --form marginBottom=0 \
    --form marginLeft=0 \
    --form marginRight=0 \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req := gotenberg.NewURLRequest("https://google.com")
    req.SetMargins(gotenberg.NoMargins)
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\URLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$request = new URLRequest('https://google.com');
$request->setMargins(Request::NO_MARGINS);
$dest = "result.pdf";
$client->store($request, $dest);
```
