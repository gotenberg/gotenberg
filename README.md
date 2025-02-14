<p align="center">
    <img src="https://user-images.githubusercontent.com/8983173/130322857-185831e2-f041-46eb-a17f-0a69d066c4e5.png" alt="Gotenberg Logo" width="150" height="150" />
    <h3 align="center">Gotenberg</h3>
    <p align="center">A Docker-powered stateless API for PDF files</p>
    <p align="center">
        <a href="https://hub.docker.com/r/gotenberg/gotenberg"><img alt="Total downloads (gotenberg/gotenberg)" src="https://img.shields.io/docker/pulls/gotenberg/gotenberg"></a>
        <a href="https://hub.docker.com/r/thecodingmachine/gotenberg"><img alt="Total downloads (thecodingmachine/gotenberg)" src="https://img.shields.io/docker/pulls/thecodingmachine/gotenberg"></a>
        <br>
        <a href="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml"><img alt="Continuous Integration" src="https://github.com/gotenberg/gotenberg/actions/workflows/continuous-integration.yml/badge.svg"></a>
        <a href="https://pkg.go.dev/github.com/gotenberg/gotenberg/v8"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/gotenberg/gotenberg.svg"></a>
        <a href="https://codecov.io/gh/gotenberg/gotenberg/branch/main/graph/badge.svg"><img alt="Code coverage" src="https://codecov.io/gh/gotenberg/gotenberg/branch/main/graph/badge.svg"></a>
    </p>
    <p align="center"><a href="https://gotenberg.dev/docs/getting-started/introduction">Documentation</a> &#183; <a href="https://gotenberg.dev/docs/getting-started/installation#live-demo-">Live Demo</a> 🔥</p>
</p>

---

**Gotenberg** provides a developer-friendly API to interact with powerful tools like Chromium and LibreOffice for converting 
numerous document formats (HTML, Markdown, Word, Excel, etc.) into PDF files, and more!

## Onebrief Gotenberg Updates

If you don't already have a GitHub personal access token, you can generate one [here](https://github.com/settings/tokens/new)

### Log in to GitHub Packages

```bash
$ export CR_PAT=TOKEN_HERE
$ echo $CR_PAT | docker login ghcr.io -u USERNAME --password-stdin
```

### Automated Build and release new version

Create a new release from the Gotenberg Github repo.

### Manual Build and release new version

```bash
$ make release
```

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
        <img src="https://user-images.githubusercontent.com/8983173/130324668-9d6e7b35-53a3-49c7-a574-38190d2bd6b0.png" alt="TheCodingMachine Logo" width="333" height="163" />
    </a>
    <a href="https://zolsec.com?utm_source=gotenberg_github&utm_medium=website" target="_blank">
        <img src="https://github.com/gotenberg/gotenberg/assets/8983173/707ccc97-a79b-4dcb-8fc8-6827366e5be3" alt="Zolsec Logo" width="333" height="163" />
    </a>
    <a href="https://pdfme.com?utm_source=gotenberg_github&utm_medium=website" target="_blank">
        <img src="https://github.com/user-attachments/assets/2a75dd40-ca18-4d34-acd5-5dd474595168" alt="pdfme Logo" width="333" height="163" />
    </a>
</p>

Sponsorships help maintaining and improving Gotenberg - [become a sponsor](https://github.com/sponsors/gulien) ❤️
