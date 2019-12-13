---
title: Markdown
---

Gotenberg provides the endpoint `/convert/markdown` for Markdown conversions.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

Markdown conversions work the same as HTML conversions.

Only difference is that you have access to the Go template function `toHTML`
in the file `index.html`. This function will convert a given markdown file to HTML.

For instance:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>My PDF</title>
  </head>
  <body>
    {{ toHTML .DirPath "file.md" }}
  </body>
</html>
```

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/markdown \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form files=@file.md \
    -o result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v7"

c := &gotenberg.Client{Hostname: "http://localhost:3000"}
index, _ := gotenberg.NewDocumentFromPath("index.html", "/path/to/file")
markdown, _ := gotenberg.NewDocumentFromPath("file.md", "/path/to/file")
req := gotenberg.NewMarkdownRequest(index, markdown)
dest := "result.pdf"
c.Store(req, dest)
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\MarkdownRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', '/path/to/file');
$markdowns = [
    DocumentFactory::makeFromPath('file.md', '/path/to/file'),
];
$request = new MarkdownRequest($index, $markdowns);
$dest = 'result.pdf';
$client->store($request, $dest);
```
