# Gotenberg Go client

A simple Go client for interacting with a Gotenberg API.

## Install

```bash
$ go get -u github.com/thecodingmachine/gotenberg
```

## Usage

```golang
import "github.com/thecodingmachine/gotenberg/pkg"

func main() {
    // HTML conversion example.
    c := &gotenberg.Client{Hostname: "http://localhost:3000"}
    req, _ := gotenberg.NewHTMLRequest("index.html")
    req.SetHeader("header.html")
    req.SetFooter("footer.html")
    req.SetAssets([]string{
        "font.woff",
        "img.gif",
        "style.css",
    })
    req.SetPaperSize(gotenberg.A4)
    req.SetMargins(gotenberg.NormalMargins)
    req.SetLandscape(false)
    dest := "foo.pdf"
    c.Store(req, dest)
}
```

For more complete usages, head to the [documentation](https://thecodingmachine.github.io/gotenberg).