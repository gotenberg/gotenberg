---
title: Merge
---

Gotenberg provides the endpoint `/merge` for merging PDFs.

It accepts `POST` requests with a `multipart/form-data` Content-Type.

## Basic

Nothing special here: you may send one or more PDF files and the API
will merge them and return the resulting PDF file.

### Guzzle

```bash
$ curl --request POST \
    --url http://localhost:3000/merge \
    --header 'Content-Type: multipart/form-data' \
    --form files=@file.pdf
    --form files=@file2.pdf
    > result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req := &gotenberg.MergeRequest{
        FilePaths: []string{
            "file.pdf",
            "file2.pdf",
        },
    }
    dest := "result.pdf"
    c.Store(req, dest)
}
```

### PHP

TODO