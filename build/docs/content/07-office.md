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
* `.html`

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

## Page ranges

You may specify the page ranges to convert.

The format is the same as the one from the print options
of LibreOffice, e.g. `1-1` or `1-4`.

> **Attention:** if more than one document, the page ranges will be
> applied for each document.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/office \
    --header 'Content-Type: multipart/form-data' \
    --form files=@document.docx \
    --form pageRanges='1-3' \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
doc, _ := gotenberg.NewDocumentFromPath("document.docx", "/path/to/file")
req := gotenberg.NewOfficeRequest(doc)
req.PageRanges("1-3")
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
$request->setPageRanges('1-3');
$dest = 'result.pdf';
$client->store($request, $dest);
```

## PDF/A-1 (ISO 19005-1:2005) compliant

You can specify the rendered output to be PDF/A-1 (ISO 19005-1:2005) compliant.

By default, the rendered output will not be not PDF/A-1 compliant

The format is the same as the one from the export filter SelectPdfVersion of 
LibreOffice, `0` PDF 1.x compliant or `1` PDF/A-1 compliant

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/office \
    --header 'Content-Type: multipart/form-data' \
    --form files=@document.docx \
    --form selectPdfVersion='1' \
    -o result.pdf
```
