<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/50009948-84b01e00-ffb8-11e8-850b-fc240382c626.png" alt="Gotenberg logo" width="250" height="250" />
</p>
<h3 align="center">Gotenberg</h3>
<p align="center">A Docker-powered stateless API for converting HTML, Markdown and Office documents to PDF.</p>
<p align="center"><a href="https://thecodingmachine.github.io/gotenberg">Documentation</a> &#183; <a href="/.github/CONTRIBUTING.md">Contributing</a></p>

---

At TheCodingMachine, we build a lot of web applications (intranets, extranets and so on) which require to generate PDF from various sources. Each time, we ended up using some well known libraries and kind of lost time by reimplementing a solution from a project to another project. Meh.

## Features

* HTML and Markdown conversions using Google Chrome headless
* Office conversions (.txt, .rtf, .docx, .doc, .odt, .pptx, .ppt, .odp and so on) using [unoconv](https://github.com/dagwieers/unoconv)
* Performance :zap:: Google Chrome and LibreOffice (unoconv) started once in the background thanks to PM2
* Failure prevention :broken_heart:: PM2 automatically restarts previous processes if they fail
* Assets :package:: send your header, footer, images, fonts, stylesheets and so on for converting your HTML and Markdown to beaufitul PDFs!
* Easily interact with the API using our [Go](https://github.com/thecodingmachine/gotenberg-go-client) and [PHP](https://github.com/thecodingmachine/gotenberg-php-client) libraries

## Quick start

Open a terminal and run the following command:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:5
```

The API is now available on your host at `http://localhost:3000`.

Head to the [documentation](https://thecodingmachine.github.io/gotenberg)
to learn how to interact with it!

## Badges

[![Docker image layers](https://images.microbadger.com/badges/image/thecodingmachine/gotenberg:5.svg)](https://microbadger.com/images/thecodingmachine/gotenberg:5)
[![Travis CI](https://travis-ci.org/thecodingmachine/gotenberg.svg?branch=master)](https://travis-ci.org/thecodingmachine/gotenberg)
[![GoDoc](https://godoc.org/github.com/thecodingmachine/gotenberg?status.svg)](https://godoc.org/github.com/thecodingmachine/gotenberg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thecodingmachine/gotenberg)](https://goreportcard.com/report/thecodingmachine/gotenberg)

---

Psst: TheCodingMachine is always looking for [talented coders](https://coders.thecodingmachine.com).
