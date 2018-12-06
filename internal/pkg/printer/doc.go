/*
Package printer contains structs which convert
a specific file type to PDF:

 // converting HTML to PDF.
 p := &printer.HTML{
	 Context: context.Background(),
	 PaperWidth: 8.27,
	 PaperHeight: 11.7,
	 Landscape: false,
 }
 p.WithLocalURL("index.html")
 if err := p.Print("result.pdf"); err != nil {
	 return err
 }

 // converting Markdown to PDF:
 // it assumes here that our template "index.html"
 // will call toHTML method to convert
 // markdown files to HTML.
 p := &printer.Markdown{
	 Context: context.Background(),
	 TemplatePath: "index.html",
	 PaperWidth: 8.27,
	 PaperHeight: 11.7,
	 Landscape: false,
 }
 if err := p.Print("result.pdf"); err != nil {
	 return err
 }

 // converting Office documents to PDF:
 // it converts each files independently and
 // then merge them.
 //
 // Also, as unoconv cannot perform
 // concurrent conversions, a lock is applied.
 p := &printer.Office{
	 Context: ctx,
	 FilePaths: []string{"document.docx", "presentation.pptx"}
 }
 if err := p.Print("result.pdf"); err != nil {
	 return err
 }

It is also able to merge a list of PDF files:

 if err := printer.Merge([]string{"foo.pdf", "bar.pdf"}, "result.pdf"); err != nil {
	 return err
 }
*/
package printer
