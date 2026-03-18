# Tester Persona

You are writing or updating tests for Gotenberg. Integration tests are the primary and preferred method.

## Integration Tests

- **Framework:** Gherkin (BDD) via [Godog](https://github.com/cucumber/godog), with `testcontainers-go` for Docker orchestration.
- **Feature files:** `test/integration/features/*.feature` — one file per endpoint or capability.
- **Test infrastructure:** `test/integration/scenario/` — Go step definitions, container management, HTTP helpers, PDF validation.
- **Entry point:** `test/integration/main_test.go` (build tag: `integration`).
- **Test data:** `test/integration/testdata/`

### How It Works

Each scenario spins up a fresh Gotenberg Docker container via testcontainers. The step definitions in `scenario/scenario.go` map Gherkin steps to Go functions. An additional `gotenberg/integration-tools` container provides PDF validation tools (`verapdf`, `pdfinfo`, `pdftotext`).

**Important:** Integration tests require a Docker image. Run `make build` before `make test-integration`.

### Selective Test Runs

Use the `TAGS` variable to run only relevant scenarios:

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="merge,split"
```

Available tags: `chromium`, `chromium-concurrent`, `chromium-convert-html`, `chromium-convert-markdown`, `chromium-convert-url`, `debug`, `health`, `libreoffice`, `libreoffice-convert`, `output-filename`, `pdfengines`, `pdfengines-convert`, `pdfengines-embed`, `embed`, `pdfengines-encrypt`, `encrypt`, `pdfengines-flatten`, `flatten`, `pdfengines-merge`, `merge`, `pdfengines-metadata`, `metadata`, `pdfengines-split`, `split`, `pdfengines-watermark`, `watermark`, `pdfengines-stamp`, `stamp`, `pdfengines-bookmarks`, `bookmarks`, `prometheus-metrics`, `root`, `version`, `webhook`, `download-from`.

Other useful flags:

```bash
make test-integration NO_CONCURRENCY=true  # Disable parallel scenarios
make test-integration PLATFORM=linux/arm64 # Force a specific platform
```

### Writing a New Integration Test

1. Create or update a `.feature` file in `test/integration/features/`.
2. Tag it appropriately (e.g., `@chromium @chromium-convert-html`).
3. If the feature requires new tag(s), add them to both the `TAGS` comment block in the `Makefile` and the "Available tags" list in this file.
4. If you create a new step definition, add it to `scenario/scenario.go`, register it in `InitializeScenario`, and update the "Available Gherkin Steps" list below.
5. Test data goes in `test/integration/testdata/`.

### Available Gherkin Steps

**Given (setup):**

- `I have a default Gotenberg container`
- `I have a Gotenberg container with the following environment variable(s):` (table: key | value)
- `I have a (webhook|static) server`

**When (action):**

- `I make a "(GET|HEAD)" request to Gotenberg at the "<endpoint>" endpoint`
- `I make a "(GET|HEAD)" request to Gotenberg at the "<endpoint>" endpoint with the following header(s):` (table: name | value)
- `I make a "(POST)" request to Gotenberg at the "<endpoint>" endpoint with the following form data and header(s):` (table: name | value | kind — where kind is `file`, `field`, or `header`)
- `I make <N> concurrent "(POST)" requests to Gotenberg at the "<endpoint>" endpoint with the following form data and header(s):` (same table format)
- `I wait for the asynchronous request to the webhook`

**Then (assertions):**

- `the response status code should be <code>`
- `the (response|webhook request) header "<name>" should be "<value>"`
- `the (response|webhook request) cookie "<name>" should be "<value>"`
- `the (response|webhook request) body should match string:` (docstring)
- `the (response|webhook request) body should contain string:` (docstring)
- `the (response|webhook request) body should match JSON:` (docstring — use `"ignore"` for dynamic values like timestamps)
- `there should be <N> PDF(s) in the (response|webhook request)`
- `there should be the following file(s) in the (response|webhook request):` (table of filenames)
- `the "<name>" PDF should have <N> page(s)`
- `the "<name>" PDF (should|should NOT) be set to landscape orientation`
- `the "<name>" PDF (should|should NOT) have the following content at page <N>:` (docstring)
- `the (response|webhook request) PDF(s) should be valid "<standard>" with a tolerance of <N> failed rule(s)` (standards: `PDF/A-1b`, `PDF/A-2b`, `PDF/A-3b`, `PDF/UA-1`, `PDF/UA-2`)
- `the (response|webhook request) PDF(s) (should|should NOT) be flatten`
- `the (response|webhook request) PDF(s) (should|should NOT) be encrypted`
- `the (response|webhook request) PDF(s) (should|should NOT) have the "<filename>" file embedded`
- `the Gotenberg container (should|should NOT) log the following entries:` (table of log substrings)
- `all concurrent response status codes should be <code>`
- `all concurrent responses should have <N> PDF(s)`

## Unit Tests

- Use **table-driven tests** for pure logic in `pkg/`.
- Mock external dependencies using the comprehensive mocks in `pkg/gotenberg/mocks.go`.
- Run with `make test-unit`.
