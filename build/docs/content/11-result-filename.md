---
title: Result filename
---

All endpoints accept a form field named `resultFilename`.

If provided, the API will return the resulting PDF file with the given filename.
Otherwise a random filename is used.

> **Attention:** this feature does not work if the form field `webhookURL` is given.

## Examples

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form resultFilename='foo.pdf'
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
req := gotenberg.NewHTMLRequest(index)
req.ResultFilename("foo.pdf")
resp, _ := c.Post(req)
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
$request->setResultFilename('foo.pdf');
$resp = $client->post($request);
```
