<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/49701110-4c8b9280-fbe8-11e8-895c-a4b9c7d2515b.png" alt="Gotenberg logo" width="250" height="250" />
</p>
<h3 align="center">Gotenberg</h3>
<p align="center">A Docker-powered stateless API for converting HTML, Markdown and Office documents to PDF.</p>

## Features

* HTML and Markdown conversions using Google Chrome headless
* Office conversions (.docx, .doc, .odt, .pptx, .ppt, .odp and so on) using [unoconv](https://github.com/dagwieers/unoconv)
* Performance :zap:: Google Chrome and Libreoffice (unoconv) started once in the background thanks to PM2
* Failure prevention :broken_heart:: PM2 automatically restarts previous processes if they fail
* Assets :package:: send your images, fonts, stylesheets and so on for converting your HTML and Markdown to beaufitul PDFs!
* Clients: easily interact with the API using `go` or `PHP`

## Quick start

Open a terminal and run the following command:

```bash
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:3
```

The API is now available on your host under `http://localhost:3000`.

Head to the [documentation](https://thecodingmachine.gotenberg.github.io)
to learn how to interact with it!

## Badges

[![Docker image layers](https://images.microbadger.com/badges/image/thecodingmachine/gotenberg:3.svg)](https://microbadger.com/images/thecodingmachine/gotenberg:3)
[![Travis CI](https://travis-ci.org/thecodingmachine/gotenberg.svg?branch=master)](https://travis-ci.org/thecodingmachine/gotenberg)
[![GoDoc](https://godoc.org/github.com/thecodingmachine/gotenberg?status.svg)](https://godoc.org/github.com/thecodingmachine/gotenberg?status.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thecodingmachine/gotenberg)](https://goreportcard.com/report/thecodingmachine/gotenberg)

---

Psst: TheCodingMachine is always looking for [talented coders](https://coders.thecodingmachine.com/).