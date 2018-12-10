---
title: Webhook
---

All endpoints accept a form field named `webhookURL`.

If provided, the API will send the resulting PDF file in a `POST` request with the `application/pdf` Content-Type
to given URL.

By doing so, your requests to the API will be done before the conversions are actually done!

## Examples

### Guzzle

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html
    --form webhookURL='http://myapp.com/webhook/'
    > result.pdf
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req := &gotenberg.HTMLRequest{
        IndexFilePath: "index.html",
        Options: &gotenberg.HTMLOptions{
            WebHookURL: "http://myapp.com/webhook/",
        },
    }
    dest := "result.pdf"
    resp, err := c.Post(req)
}
```

### PHP

TODO