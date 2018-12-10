/*
Package gotenberg is a Go client for
interacting with a Gotenberg API.

For instance, if you want to convert HTML:

 import "github.com/thecodingmachine/gotenberg/pkg"

 func main() {
     // HTML conversion example.
     c := &gotenberg.Client{Hostname: "http://localhost:3000"}
     req := &gotenberg.HTMLRequest{
         IndexFilePath: "index.html",
         AssetFilePaths: []string{
             "style.css",
             "img.png",
         },
         Options: &gotenberg.HTMLOptions{
             HeaderFilePath: "header.html",
             FooterFilePath: "footer.html",
             PaperSize:      gotenberg.A4,
             PaperMargins:   gotenberg.NormalMargins,
         },
     }
     dest := "foo.pdf"
     c.Store(req, dest)
 }

For more complete usages, head to the https://thecodingmachine.gotenberg.github.io.
*/
package gotenberg
