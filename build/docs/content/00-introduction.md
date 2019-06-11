---
title: Introduction
---

[Gotenberg](https://github.com/thecodingmachine/gotenberg/) is a Docker-powered stateless API for converting HTML, Markdown and Office documents to PDF.

* HTML and Markdown conversions using Google Chrome headless
* Office conversions (.txt, .rtf, .docx, .doc, .odt, .pptx, .ppt, .odp and so on) using [unoconv](https://github.com/dagwieers/unoconv)
* Performance: Google Chrome and LibreOffice (unoconv) started once in the background thanks to PM2
* Failure prevention: PM2 automatically restarts previous processes if they fail
* Assets: send your header, footer, images, fonts, stylesheets and so on for converting your HTML and Markdown to beaufitul PDFs!
* Easily interact with the API using our [Go](https://github.com/thecodingmachine/gotenberg-go-client) and [PHP](https://github.com/thecodingmachine/gotenberg-php-client) libraries