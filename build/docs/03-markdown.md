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
    --form files=@index.html
    --form files=@file.md
    > result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req := &gotenberg.MarkdownRequest{
        IndexFilePath: "index.html",
        MarkdownFilePaths: []string{
            "file.md",
        },
    }
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\MarkdownRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$markdowns = [
    DocumentFactory::makeFromPath('file.md', 'file.md'),
];
$request = new MarkdownRequest($index, $markdowns);
$dirPath = "/foo";
$filename = $client->store($request, $dirPath);
```
