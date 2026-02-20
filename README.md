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
    <p align="center"><a href="https://gotenberg.dev/docs/getting-started/introduction">Read the Documentation</a> &#183; <a href="https://gotenberg.dev/docs/getting-started/installation#live-demo-">Try the Live Demo</a> 🔥</p>
</p>

---

**Gotenberg** is a containerized API that abstracts the complexity of PDF conversion.

It provides a `multipart/form-data` interface for interacting with powerful engines like Chromium and LibreOffice.
Instead of managing heavy dependencies, browser versions, or fonts in your own backend, simply send your files to
Gotenberg and get a PDF in return.

## Quick Start

Open a terminal and run the following command:

```bash
docker run --rm -p 3000:3000 gotenberg/gotenberg:8
```

With the API running at `http://localhost:3000`, you are now ready to head
to the **[Full Documentation](https://gotenberg.dev/docs/getting-started/introduction)** to discover how to convert URLs,
local files, inject custom CSS, merge PDFs, and more.

## Sponsors

Open-source development takes a significant amount of time, energy, and dedication. If Gotenberg helps streamline your
workflow or powers your business, please consider supporting its continuous improvement by [**becoming a sponsor**](https://github.com/sponsors/gulien)! ❤️

**GitHub Sponsors**

- [TheCodingMachine](https://thecodingmachine.com/)
- [pdfme](https://pdfme.com/)

**Powered By**

- [Docker](https://docs.docker.com/docker-hub/repos/manage/trusted-content/dsos-program/)
- [JetBrains](https://www.jetbrains.com/community/opensource/)
