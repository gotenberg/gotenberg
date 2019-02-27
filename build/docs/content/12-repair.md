---
title: PDF Repair
---

Gotenberg provides the endpoint `/repair` for trying to repair damaged PDF.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

You may send a single PDF documents.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/repair \
    --header 'Content-Type: multipart/form-data' \
    --form files=@broken.pdf \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewPdfRepairRequest("broken.pdf", "broken.pdf")
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\PdfRepairRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$file = DocumentFactory::makeFromPath('broken.pdf', 'broken.pdf');
$request = new PdfRepairRequest($file);
$dest = "result.pdf";
$client->store($request, $dest);
```
