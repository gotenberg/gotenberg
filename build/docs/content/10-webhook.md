---
title: Webhook
---

All endpoints accept a form field named `webhookURL`.

If provided, the API will send the resulting PDF file in a `POST` request with the `application/pdf` Content-Type
to given URL.

By doing so, your requests to the API will be over before the conversions are actually done!

## Examples

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form webhookURL='http://myapp.com/webhook/'
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.WebhookURL("http://myapp.com/webhook/")
    resp, _ := c.Post(req)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$request->setWebhookURL('http://myapp.com/webhook/');
$resp = $client->post($request);
```

## Timeout

If a `webhookURL` is provided, you may also send a form field named `webhookURLTimeout`.

The API will wait the given **seconds** before it considers the sending of the resulting PDF to be unsucessful.

It takes a float as value (e.g `2.5` for 2.5 seconds).

> You may also define this value globally: see the [environment variables](#environment_variables.default_webhook_url_timeout) section.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --form files=@index.html \
    --form webhookURL='http://myapp.com/webhook/' \
    --form webhookURLTimeout=2.5
```

### Go

```golang
import "github.com/thecodingmachine/gotenberg-go-client/v6"

func main() {
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.WebhookURL("http://myapp.com/webhook/")
    req.WebhookURLTimeout(2.5)
    resp, _ := c.Post(req)
}
```

### PHP

```php
use TheCodingMachine\Gotenberg\Client;
use TheCodingMachine\Gotenberg\DocumentFactory;
use TheCodingMachine\Gotenberg\HTMLRequest;

$client = new Client('http://localhost:3000', new \Http\Adapter\Guzzle6\Client());
$index = DocumentFactory::makeFromPath('index.html', 'index.html');
$request = new HTMLRequest($index);
$request->setWebhookURL('http://myapp.com/webhook/');
$request->setWebhookURLTimeout(2.5);
$resp = $client->post($request);
```

## Custom HTTP headers

You may send your own HTTP headers to the `webhookURL`.

For instance, by adding the HTTP header `Gotenberg-Webhookurl-Your-Header` to your request,
the API will send a request to the `webhookURL` with the HTTP header `Your-Header`.

> **Attention:** the API uses a canonical format for the HTTP headers:
> it transforms the first
> letter and any letter following a hyphen to upper case;
> the rest are converted to lowercase. For example, the
> canonical key for `accept-encoding` is `Accept-Encoding`.

### cURL

```bash
$ curl --request POST \
    --url http://localhost:3000/convert/html \
    --header 'Content-Type: multipart/form-data' \
    --header 'Gotenberg-Webhookurl-Your-Header: Foo' \
    --form files=@index.html \
    --form webhookURL='http://myapp.com/webhook/'
```

### Go

// TODO

### PHP

// TODO