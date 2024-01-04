<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/130322857-185831e2-f041-46eb-a17f-0a69d066c4e5.png" alt="Gotenberg Logo" width="150" height="150" />
    <h3 align="center">Gotenberg</h3>
    <p align="center">A Docker-powered stateless API for PDF files</p>
    <p align="center"><a href="https://gotenberg.dev/docs/getting-started/introduction">Documentation</a> &#183; <a href="https://gotenberg.dev/docs/getting-started/installation#live-demo-">Live Demo</a> 🔥</p>
</p>

---

Gotenberg provides a developer-friendly API to interact with powerful tools like Chromium and LibreOffice for converting 
numerous document formats (HTML, Markdown, Word, Excel, etc.) into PDF files, and more!

## Quick Start

Open a terminal and run the following command:

```
docker run --rm -p 3000:3000 gotenberg/gotenberg:8
```

Alternatively, using the historic Docker repository from our sponsor [TheCodingMachine](https://www.thecodingmachine.com):

```
docker run --rm -p 3000:3000 thecodingmachine/gotenberg:8
```

The API is now available on your host at http://localhost:3000.

Head to the [documentation](https://gotenberg.dev/docs/getting-started/introduction) to learn how to interact with it 🚀

## Sponsors

<p align="center">
    <a href="https://thecodingmachine.com">
        <img src="https://user-images.githubusercontent.com/8983173/130324668-9d6e7b35-53a3-49c7-a574-38190d2bd6b0.png" alt="TheCodingMachine Logo" width="429" height="210" />
    </a>
</p>

## Badges

[![Docker pulls](https://img.shields.io/docker/pulls/gotenberg/gotenberg)](https://hub.docker.com/r/gotenberg/gotenberg)
[![Docker pulls](https://img.shields.io/docker/pulls/thecodingmachine/gotenberg)](https://hub.docker.com/r/thecodingmachine/gotenberg)
[![Continuous Integration](https://github.com/gotenberg/gotenberg/actions/workflows/continuous_integration.yml/badge.svg)](https://github.com/gotenberg/gotenberg/actions/workflows/continuous_integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/gotenberg/gotenberg.svg)](https://pkg.go.dev/github.com/gotenberg/gotenberg/v8)
[![Codecov](https://codecov.io/gh/gotenberg/gotenberg/branch/main/graph/badge.svg)](https://codecov.io/gh/gotenberg/gotenberg)
