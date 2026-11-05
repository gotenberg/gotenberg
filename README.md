<p align="center">
    <img src="https://raw.githubusercontent.com/gotenberg/art/main/logo.png" alt="Gotenberg Logo" width="150" height="150" />
    <h3 align="center">Gotenberg</h3>
    <p align="center">A Docker-based API for PDF</p>
    <p align="center">
        <a href="https://hub.docker.com/r/gotenberg/gotenberg"><img alt="Total downloads (gotenberg/gotenberg)" src="https://img.shields.io/docker/pulls/gotenberg/gotenberg"></a>
        <a href="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml"><img alt="Continuous Integration" src="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml/badge.svg"></a>
        <a href="https://pkg.go.dev/github.com/gotenberg/gotenberg/v9"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/gotenberg/gotenberg.svg"></a>
    </p>
    <p align="center">
        <a href="https://trendshift.io/repositories/2996"><img src="https://trendshift.io/api/badge/repositories/2996" alt="gotenberg%2Fgotenberg | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>
    </p>
    <p align="center">
        <a href="https://gotenberg.dev/docs"><strong>Documentation</strong></a> &#183;
        <a href="https://gotenberg.dev/docs/getting-started/installation#live-demo"><strong>Live Demo</strong></a> &#183;
        <a href="https://github.com/gotenberg/gotenberg/discussions"><strong>Community</strong></a>
    </p>
</p>

---

**Gotenberg** is a Docker-based API for PDF. It converts web pages and office documents to PDF, then merges, stamps and signs them, makes them archival or accessible, and delivers them. Trusted in production by thousands of companies, and adopted by notable open-source projects.

Send your files via `multipart/form-data`, get a PDF back. No need to manage Chromium, LibreOffice, or fonts yourself.

## Quick Start

```bash
docker run --rm -p 3000:3000 gotenberg/gotenberg:9
```

Convert a URL to PDF:

```bash
curl \
  --request POST http://localhost:3000/forms/chromium/convert/url \
  --form url=https://sparksuite.github.io/simple-html-invoice-template/ \
  -o invoice.pdf
```

Make it a PDF/A-2b in the same request:

```bash
curl \
  --request POST http://localhost:3000/forms/chromium/convert/url \
  --form url=https://sparksuite.github.io/simple-html-invoice-template/ \
  --form 'stages=[{"op":"remediate","profile":"PDF/A-2b"}]' \
  -o invoice.pdf
```

Coming from Gotenberg 8? `gotenberg/gotenberg:8` stays available, and the [upgrade guide](https://gotenberg.dev/docs/getting-started/upgrade) maps every route, field and flag to its Gotenberg 9 counterpart.

## Features

- **HTML, URL, Markdown to PDF** via Headless Chromium
- **Office documents to PDF** via LibreOffice (100+ formats)
- **Merge, split, select, rotate, stamp** PDFs, and **fill or flatten** their forms
- **PDF/A and PDF/UA**: validate, then remediate until the PDF conforms, Factur-X included
- **Sign, timestamp, encrypt, redact, optimize** PDFs
- **Render pages as images**, and screenshots of URLs and HTML
- **Read and write metadata, bookmarks and attachments**

See the [full documentation](https://gotenberg.dev/docs).

## Contributing

- Contributing guide: [CONTRIBUTING.md](CONTRIBUTING.md)
- Bug reports: [GitHub Issues](https://github.com/gotenberg/gotenberg/issues)
- Questions and feedback: [GitHub Discussions](https://github.com/gotenberg/gotenberg/discussions)
- Security vulnerabilities: [SECURITY.md](SECURITY.md)

## Sponsors

If Gotenberg powers your workflow or your business, consider [**becoming a sponsor**](https://github.com/sponsors/gulien).

**Historic & GitHub Sponsors**

- [TheCodingMachine](https://thecodingmachine.com/)
- [pdfme](https://pdfme.com/)
- [PDFBolt](https://pdfbolt.com)
- [FileToPDF.dev](https://filetopdf.dev)

**Powered By**

- [Docker](https://docs.docker.com/docker-hub/repos/manage/trusted-content/dsos-program/)
- [JetBrains](https://www.jetbrains.com/community/opensource/)
