---
title: Timeout
---

All endpoints accept a form field named `waitTimeout`.

The API will wait the given **seconds** before it considers the conversion to be unsucessful.
If unsucessful, it returns a `504` HTTP code.

It takes a float as value (e.g `2.5` for 2.5 seconds).

> You may also define this value globally: see the [environment variables](#environment_variables.default_wait_timeout) section.

## Examples

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form waitTimeout=2.5
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.WaitTimeout(2.5)
    resp, _ := c.Post(req)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;
use TheCodingMachine\Gotenberg\Request;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$request->setWaitTimeout(2.5);
$dest = "result.pdf";
$client->store($request, $dest);
```
