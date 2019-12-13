---
title: Merge
---

Gotenberg provides the endpoint `/merge` for merging PDFs.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

Nothing fancy here: you may send one or more PDF files and the API
will merge them and return the resulting PDF file.

> **Attention:** Gotenberg merges the PDF files alphabetically.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/merge \
    --header 'Content-Type: multipart/form-data' \
    --form files=@file.pdf \
    --form files=@file2.pdf \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
pdf, _ := gotenberg.NewDocumentFromPath("file.pdf", "/path/to/file")
pdf2, _ := gotenberg.NewDocumentFromPath("file2.pdf", "/path/to/file")
req := gotenberg.NewMergeRequest(pdf, pdf2)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\MergeRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$files = [
    DocumentFactory::makeFromPath('file.pdf', '/path/to/file'),
    DocumentFactory::makeFromPath('file2.pdf', '/path/to/file'),
];
$request = new MergeRequest($files);
$dest = 'result.pdf';
$client->store($request, $dest);
```
