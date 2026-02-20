<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/130322857-185831e2-f041-46eb-a17f-0a69d066c4e5.png" alt="Gotenberg Logo" width="150" height="150" />
    <h3 align="center">Gotenberg</h3>
    <p align="center">A containerized API for seamless PDF conversion</p>
    <p align="center">
        <a href="https://hub.docker.com/r/gotenberg/gotenberg"><img alt="Total downloads (gotenberg/gotenberg)" src="https://img.shields.io/docker/pulls/gotenberg/gotenberg"></a>
        <a href="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml"><img alt="Continuous Integration" src="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml/badge.svg"></a>
        <a href="https://pkg.go.dev/github.com/gotenberg/gotenberg/v8"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/gotenberg/gotenberg.svg"></a>
    </p>
    <p align="center">
        <a href="https://trendshift.io/repositories/2996"><img src="https://trendshift.io/api/badge/repositories/2996" alt="gotenberg%2Fgotenberg | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>
    </p>
    <p align="center"><a href="https://gotenberg.dev/docs/getting-started/introduction">Documentation</a> &#183; <a href="https://gotenberg.dev/docs/getting-started/installation#live-demo-">Live Demo</a> 🔥</p>
</p>

---

**Gotenberg** provides a developer-friendly API to interact with powerful tools like Chromium and LibreOffice for converting
numerous document formats (HTML, Markdown, Word, Excel, etc.) into PDF files, and more!

## Quick Start

Open a terminal and run the following command:

```bash
docker run --rm -p 3000:3000 gotenberg/gotenberg:8
```

With the API running at `http://localhost:3000`, you can immediately convert a URL to a PDF:

```bash
curl \
  --request POST http://localhost:3000/forms/chromium/convert/url \
  --form url=https://sparksuite.github.io/simple-html-invoice-template/ \
  -o invoice.pdf
```

**Read the [Full Documentation](https://gotenberg.dev/docs/getting-started/introduction)** to discover how to convert local
files, inject custom CSS, merge PDFs, and more.

## Sponsors

Open-source development takes time and effort. Support the continuous improvement of Gotenberg by [**becoming a sponsor**](https://github.com/sponsors/gulien)! ❤️

**GitHub Sponsors**

- [TheCodingMachine](https://thecodingmachine.com/)
- [pdfme](https://pdfme.com/)

**Powered By**

- [Docker](https://docs.docker.com/docker-hub/repos/manage/trusted-content/dsos-program/)
- [JetBrains](https://www.jetbrains.com/community/opensource/)
