<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/38133342-11df3bd8-340f-11e8-9fe4-50baecdceeca.png" alt="Gotenberg's logo" width="250" height="250" />
</p>
<h3 align="center">Gotenberg</h3>
<p align="center">A stateless API for converting Markdown files, HTML files and Office documents to PDF</p>
<p align="center">
    <a href="https://microbadger.com/images/thecodingmachine/gotenberg:{{ .Orbit.Branch }}">
        <img src="https://images.microbadger.com/badges/version/thecodingmachine/gotenberg:{{ .Orbit.Branch }}.svg" alt="MicroBadger version">
    </a>
    <a href="https://microbadger.com/images/thecodingmachine/gotenberg:{{ .Orbit.Branch }}">
        <img src="https://images.microbadger.com/badges/image/thecodingmachine/gotenberg:{{ .Orbit.Branch }}.svg" alt="MicroBadger layers">
    </a>
    <a href="https://travis-ci.org/thecodingmachine/gotenberg">
        <img src="https://travis-ci.org/thecodingmachine/gotenberg.svg?branch={{ .Orbit.Branch }}" alt="Travis CI">
    </a>
    <a href="https://godoc.org/github.com/thecodingmachine/gotenberg">
        <img src="https://godoc.org/github.com/thecodingmachine/gotenberg?status.svg" alt="GoDoc">
    </a>
    <a href="https://goreportcard.com/report/thecodingmachine/gotenberg">
        <img src="https://goreportcard.com/badge/github.com/thecodingmachine/gotenberg" alt="Go Report Card">
    </a>
    <a href="https://codecov.io/gh/thecodingmachine/gotenberg/branch/{{ .Orbit.Branch }}">
        <img src="https://codecov.io/gh/thecodingmachine/gotenberg/branch/{{ .Orbit.Branch }}/graph/badge.svg" alt="Codecov">
    </a>
</p>

---

# Menu

* [Quick start](#quick-start)
* [API](#api)
* [Custom implementation](#custom-implementation)
* [Scalability](#scalability)

## Quick start

```sh
$ docker run --rm -p 3000:3000 thecodingmachine/gotenberg:{{ .Orbit.Branch }}
```

The API is now available through `http://127.0.0.1:3000`.

## API

## Custom implementation

## Scalability

---

Would you like to update this documentation ? Feel free to open an [issue](../../issues).