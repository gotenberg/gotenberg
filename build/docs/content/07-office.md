---
title: Office
---

Gotenberg provides the endpoint `/convert/office` for Office document conversions.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

You may send one or more Office documents. Following file extensions are accepted:

* `.txt`
* `.rtf`
* `.fodt`
* `.doc`
* `.docx`
* `.odt`
* `.xls`
* `.xlsx`
* `.ods`
* `.ppt`
* `.pptx`
* `.odp`

All files will be merged into a single resulting PDF.

> **Attention:** Gotenberg merges the PDF files alphabetically.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/office \
    --header 'Content-Type: multipart/form-data' \
    --form files=@document.docx \
    --form files=@document2.docx \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
doc, _ := gotenberg.NewDocumentFromPath("document.docx", "/path/to/file")
doc2, _ := gotenberg.NewDocumentFromPath("document2.docx", "/path/to/file")
req := gotenberg.NewOfficeRequest(doc, doc2)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\OfficeRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$files = [
    DocumentFactory::makeFromPath('document.docx', '/path/to/file'),
    DocumentFactory::makeFromPath('document2.docx', '/path/to/file'),
];
$request = new OfficeRequest($files);
$dest = 'result.pdf';
$client->store($request, $dest);
```

## Orientation

You may also customize the resulting PDF format.

By default, it will be rendered with `portrait` orientation.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/office \
    --header 'Content-Type: multipart/form-data' \
    --form files=@document.docx \
    --form landscape=true \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
doc, _ := gotenberg.NewDocumentFromPath("document.docx", "/path/to/file")
req := gotenberg.NewOfficeRequest(doc)
req.Landscape(true)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\OfficeRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$files = [
    DocumentFactory::makeFromPath('document.docx', '/path/to/file'),
];
$request = new OfficeRequest($files);
$request->setLandscape(true);
$dest = 'result.pdf';
$client->store($request, $dest);
```
