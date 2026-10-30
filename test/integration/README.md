# Integration Tests

Gherkin (BDD) via [Godog](https://github.com/cucumber/godog), with `testcontainers-go` for Docker orchestration.

- Feature files: `test/integration/features/*.feature`, one per endpoint or capability.
- Step definitions: `test/integration/scenario/` (container management, HTTP helpers, PDF validation).
- Entry point: `test/integration/main_test.go` (build tag: `integration`).
- Test data: `test/integration/testdata/`.

Each scenario spins up a fresh Gotenberg Docker container via testcontainers. The PDF assertions read the produced documents with [litho](https://github.com/gotenberg/litho), in process: page count, size, text, tagging, bookmarks, metadata, attachments, permissions, signatures, and PDF/A and PDF/UA conformance.

Run `make build` before `make test-integration`. Integration tests require a Docker image.

## Selective runs

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="pdf-merge,pdf-split"
make test-integration CONCURRENCY=1           # one scenario at a time (0, the default, runs one per CPU)
make test-integration PLATFORM=linux/arm64    # force a specific platform
make test-integration KEEP_FAILED=/tmp/failed # keep the outputs of failed scenarios
```

The scenarios work in a temporary directory the run removes when it ends, on an interrupt (Ctrl+C) included. `KEEP_FAILED` moves the directory of each failed scenario aside first, and the run prints where.

Available tags:

| Group       | Tags                                                                                                                                                                                                                                                                         |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Chromium    | `chromium`, `chromium-concurrent`, `chromium-convert-html`, `chromium-image-deduplication`, `chromium-convert-markdown`, `chromium-convert-url`, `chromium-screenshot-html`, `chromium-screenshot-markdown`, `chromium-screenshot-url`, `chromium-ssrf`                      |
| LibreOffice | `libreoffice`, `libreoffice-convert`, `libreoffice-ssrf`                                                                                                                                                                                                                     |
| PDF         | `pdf`, `pdf-pipe`, `pdf-merge`, `pdf-split`, `pdf-pages`, `pdf-rotate`, `pdf-stamp`, `pdf-metadata`, `pdf-bookmarks`, `pdf-form`, `pdf-attach`, `pdf-encrypt`, `pdf-remediate`, `factur-x`, `pdf-sign`, `pdf-optimize`, `pdf-redact`, `pdf-colors`, `pdf-render`, `pdf-read` |
| Infra       | `health`, `debug`, `root`, `version`, `output-filename`, `webhook`, `download-from`, `api-limits`, `api-archives`                                                                                                                                                            |

## Writing a new test

1. Create or update a `.feature` file in `test/integration/features/`.
2. Tag it appropriately (e.g., `@chromium @chromium-convert-html`).
3. For new tags, add them to both the `TAGS` comment block in the `Makefile` and the table above.
4. For new step definitions, add the function to `scenario/scenario.go` (or `scenario/pdf.go` for PDF assertions), register it in `InitializeScenario`, and add the step pattern to the step reference below.
5. Test data goes in `test/integration/testdata/`.
6. `I have a default Gotenberg container` reuses a container an earlier scenario left, one scenario at a time. A scenario that reads the container's logs, stops it, or sends concurrent or sequential requests gets its own. Tag a scenario `@isolated` to give it its own container for another reason. A container with environment variables is never reused.

## Step reference

### Given (setup)

- `I have a default Gotenberg container`
- `I have a Gotenberg container with the following environment variable(s):` (table: key | value)
- `I have a (webhook|static) server`: one helper server per scenario, whichever word is used; a second step reuses it. Besides the webhook and file routes, it answers RFC 3161 timestamp requests at `{helper}/tsa`

### When (action)

- `I make a "(GET|HEAD)" request to Gotenberg at the "<endpoint>" endpoint`
- `I make a "(GET|HEAD)" request to Gotenberg at the "<endpoint>" endpoint with the following header(s):` (table: name | value)
- `I make a "(POST)" request to Gotenberg at the "<endpoint>" endpoint with the following form data and header(s):` (table: name | value | kind, where kind is `file`, `field`, or `header`)
- `I make <N> concurrent "(POST)" requests to Gotenberg at the "<endpoint>" endpoint with the following form data and header(s):` (same table format)
- `I make <N> sequential "(POST)" requests to Gotenberg at the "<endpoint>" endpoint, probing "<probe endpoint>" after each, with the following form data and header(s):` (same table format)
- `I wait for the asynchronous request to the webhook` (fails after 90s)
- `I stop the Gotenberg container` (SIGTERM, then expects a clean exit)

### Then (assertions)

- `the response status code should be <code>`
- `the (response|webhook request|file request|server request) header "<name>" should be "<value>"` (the helper server's last request of that kind: `webhook request` is a webhook delivery, `file request` a file served from `/static`, `server request` a page served from `/html` or `/redirect-to-private`)
- `the webhook request header "<name>" should carry trace id "<trace id>"` (a W3C `traceparent` header)
- `the (response|webhook request|file request|server request) cookie "<name>" should be "<value>"`
- `the (response|webhook request) body should match string:` (docstring)
- `the (response|webhook request) body should contain string:` (docstring)
- `the (response|webhook request) body should contain the following line(s):` (docstring; each line must appear, in any order)
- `the (response|webhook request) body should match JSON:` (docstring, use `"ignore"` for dynamic values)
- `the webhook event should match JSON:` (docstring, use `"ignore"` for dynamic values; polls for up to 5s)
- `there should be <N> PDF(s) in the (response|webhook request)`
- `there should be the following file(s) in the (response|webhook request):` (table of filenames)
- `the "<name>" PDF should have <N> page(s)`
- `the "<name>" PDF (should|should NOT) be set to landscape orientation`
- `the "<name>" PDF (should|should NOT) be tagged`
- `the "<name>" PDF (should|should NOT) have the following content at page <N>:` (docstring)
- `the "<name>" PDF (should|should NOT) have content matching "<regexp>" at page <N>`
- `the (response|webhook request) PDF(s) should be valid "<standard>" with a tolerance of <N> failed rule(s)` (standards: `PDF/A-1b`, `PDF/A-2b`, `PDF/A-3b`, `PDF/A-4`, `PDF/UA-1`, `PDF/UA-2`; counts distinct failed rules, as veraPDF does)
- `the (response|webhook request) PDF(s) (should|should NOT) be flatten`
- `the (response|webhook request) PDF(s) (should|should NOT) be encrypted`
- `the (response|webhook request) PDF(s) (should|should NOT) allow "<action>"` (actions: `printing`, `copying`, `modifying`, `annotating`)
- `the (response|webhook request) PDF(s) (should|should NOT) have the "<filename>" file embedded`
- `the (response|webhook request) PDF(s) should have the "<filename>" file embedded with relationship "<relationship>"`
- `the (response|webhook request) PDF(s) should declare Factur-X XMP with conformance level "<level>"`
- `the (response|webhook request) PDF(s) (should|should NOT) be signed`
- `the "<name>" PDF should have the "<key>" metadata set to "<value>"` (keys: `Title`, `Author`, `Subject`, `Keywords`, `Creator`, `Producer`)
- `the "<name>" PDF should have the form field "<field>" set to "<value>"`
- `the "<name>" PDF page <N> should be rotated by <angle> degrees`
- `the "<name>" PDF first page should be <width>x<height> points` (one point of tolerance)
- `the "<name>" PDF (should|should NOT) have a document outline`
- `the "<name>" PDF should have <N> image(s)`
- `the "<name>" PDF should reuse <N> image XObject(s) across <P> page(s)` (checks image appearances on every page and distinct PDF object IDs)
- `the Gotenberg container (should|should NOT) log the following entries:` (table of log substrings)
- `all concurrent response status codes should be <code>`
- `all concurrent responses should have <N> PDF(s)`
- `at least one concurrent response status code should be <code>`
- `the (webhook request|file request|server request) method should be "<method>"`
- `all sequential response status codes should be <code>`
- `all probe response status codes should be <code>`
- `the "<name>" image should be <width>x<height> pixels`
- `the "<name>" image pixel at <x>,<y> should be "<color>"`
- `the "<name>" file size should be (less|greater) than <N> KB`
