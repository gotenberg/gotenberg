# Operational Guidelines for Gotenberg

You are working on **Gotenberg**, a Docker-based API for converting documents to PDF. It is a widely used production dependency. Stability and backward compatibility are paramount. When in doubt about whether a change is breaking, flag it rather than assuming it's safe.

## Mandatory Workflow

Every task MUST follow these five steps in order. Do not skip any step.

### Step 1 — Plan

Before writing any code, produce a plan that covers:

- **Problem statement**: What needs to change and why.
- **Proposed solution**: The recommended approach with enough detail to implement (files to modify, interface changes, pipeline positioning, form fields, etc.).
- **Alternatives considered**: At least one alternative approach when pertinent, with a brief explanation of why the proposed solution is preferred.
- **Scope**: List every file that will be created or modified.
- **Testing strategy**: Which integration test tags will be affected, what new scenarios are needed, and whether unit tests are required.

Present the plan to the user and wait for approval before proceeding to Step 2. If the user provides a plan, validate it against the codebase and flag any issues before implementing.

### Step 2 — Implement

Implement the approved plan following the coding standards and patterns described in this document. After implementation, verify the build compiles (`go build ./...`).

### Step 3 — Test

Write or update tests based on the plan's testing strategy:

- **Integration tests** (primary): Gherkin scenarios in `test/integration/features/`. See the [Integration Tests](#integration-tests) section.
- **Unit tests** (when applicable): Table-driven tests in `*_test.go` files using mocks from `pkg/gotenberg/mocks.go`.

### Step 4 — Review

Self-review the implementation against the [Review Checklist](#review-checklist). Fix any issues found before presenting the result to the user.

### Step 5 — Commit

Present the review to the user and **wait for explicit approval**. Do NOT commit until the user confirms. Once approved, create a commit following the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>
```

Stage only the files related to the change. Do not use `git add -A` or `git add .`.

---

## Core Principles

- **Backward compatibility is law.** Never modify existing CLI flags, environment variables, or API form fields unless explicitly instructed to perform a breaking change. Flag any breaking change immediately.
- **Defensive programming.** Assume input is malformed. Handle errors explicitly. Never panic.
- **Atomic commits.** One feature or fix per PR. Isolate refactoring from feature work.
- **Idiomatic Go.** Follow "Effective Go" principles. All exported symbols must have GoDoc comments starting with their name.

## Project Layout

```
cmd/gotenberg/       → Entry point only (wiring/startup). No business logic.
pkg/gotenberg/       → Core module system, interfaces, utilities, mocks.
pkg/modules/         → Feature modules (api, chromium, libreoffice, pdfengines, etc.).
pkg/standard/        → Wires all standard modules together via imports.
test/integration/    → Gherkin feature files + Go test infrastructure.
build/               → Dockerfile, fonts, Chromium config.
.bruno/              → Bruno API collection (mirrors every route).
```

Key interfaces live in `pkg/gotenberg/` — `Module`, `Provisioner`, `Validator`, `Debuggable`. Every module implements `Descriptor()` and self-registers. When adding features, determine if they belong in an existing module or require a new one.

## Codebase Navigation

- Start with `pkg/gotenberg/` for core interfaces and `pkg/modules/` for feature implementations.
- The integration test infrastructure in `test/integration/scenario/` is well-structured — read `scenario.go` and `containers.go` to understand the Gherkin step definitions before writing new tests.
- Mocks for all major interfaces are in `pkg/gotenberg/mocks.go` — use them for unit tests rather than creating new ones.
- Import ordering is enforced: standard library, third-party, then `github.com/gotenberg/gotenberg/v8` — separated by blank lines.
- When making changes, run only the relevant integration test tag rather than the full suite (40min timeout).

---

## Makefile — the Only Build Interface

All build and verification tasks go through the Makefile. Do not run `go` commands directly unless debugging a specific package.

| Command                 | Purpose                                                       | When to use                                                            |
| ----------------------- | ------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `make build`            | Build the Docker image                                        | Before integration tests, or to verify compilation                     |
| `make run`              | Run a Gotenberg container locally                             | Manual testing. Flags are configured via `.env` and Makefile variables |
| `make fmt`              | Format Go code (`go fix`, `golangci-lint fmt`, `go mod tidy`) | Before every commit                                                    |
| `make lint`             | Lint Go code (strict `.golangci.yml` config)                  | Before every commit. Zero errors permitted                             |
| `make lint-prettier`    | Lint non-Go files (Markdown, YAML, etc.) with Prettier        | Before every commit                                                    |
| `make prettify`         | Format non-Go files (Markdown, YAML, etc.) with Prettier      | Before every commit                                                    |
| `make test-unit`        | Run unit tests (`go test -race ./...`)                        | After code changes to `pkg/`                                           |
| `make test-integration` | Run integration tests (Gherkin/Godog, 40min timeout)          | After any feature or route change                                      |
| `make godoc`            | Serve GoDoc at `localhost:6060`                               | To verify documentation                                                |

## Module System

Gotenberg uses a self-registering module architecture inspired by CaddyServer. Each module:

- Lives in `pkg/modules/<name>/`
- Implements the `gotenberg.Module` interface (at minimum `Descriptor()`)
- May also implement `gotenberg.Provisioner`, `gotenberg.Validator`, or `gotenberg.Debuggable`
- Self-registers via `init()` and is wired through `pkg/standard/`

When adding a feature, first determine if it belongs in an existing module. Only create a new module if the feature represents a genuinely separate concern.

## Commit Convention

Commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`. The scope should match the module or area of the change (e.g., `chromium`, `pdfengines`, `api`).

## Coding Patterns

- **Error handling:** Always wrap errors with context using `fmt.Errorf("description: %w", err)`. Never swallow errors silently.
- **Import ordering:** Enforced by `gci` — standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.
- **Mocks:** Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests.
- **No business logic in `cmd/`:** The `cmd/gotenberg/` package is strictly for wiring and startup.

## Adding PDF Engine Features

When adding a new PDF engine capability (e.g., bookmarks, watermark, stamp, embed), you must update the Makefile to include the corresponding engine list variable and flag. Every `--pdfengines-*-engines` flag registered in `pkg/modules/pdfengines/pdfengines.go` must have a matching entry in the Makefile:

1. **Add a variable** in the Makefile's variable block (around line 60-70):
   ```makefile
   PDFENGINES_<FEATURE>_ENGINES=<default engines>
   ```
2. **Add the flag** in the Makefile's command args block (around line 140-155):
   ```makefile
   --pdfengines-<feature>-engines=$(PDFENGINES_<FEATURE>_ENGINES) \
   ```

The default value should match what is defined in `pdfengines.go`'s `fs.StringSlice(...)` call for that flag.

---

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

Available tags: `chromium`, `chromium-concurrent`, `chromium-convert-html`, `chromium-convert-markdown`, `chromium-convert-url`, `debug`, `health`, `libreoffice`, `libreoffice-convert`, `output-filename`, `pdfengines`, `pdfengines-convert`, `pdfengines-embed`, `embed`, `pdfengines-encrypt`, `encrypt`, `pdfengines-flatten`, `flatten`, `pdfengines-merge`, `merge`, `pdfengines-metadata`, `metadata`, `pdfengines-split`, `split`, `pdfengines-watermark`, `watermark`, `pdfengines-stamp`, `stamp`, `pdfengines-bookmarks`, `bookmarks`, `pdfengines-rotate`, `rotate`, `prometheus-metrics`, `root`, `version`, `webhook`, `download-from`.

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

---

## Review Checklist

### Backward Compatibility

- [ ] No existing CLI flags renamed or removed
- [ ] No existing environment variables renamed or removed
- [ ] No existing API form fields renamed or removed
- [ ] No existing HTTP endpoints changed or removed
- [ ] No changes to default values that alter existing behavior

If any of these are violated, the change **must** be flagged as a breaking change.

### Linting Standards

The `.golangci.yml` enforces strict rules including: `gosec`, `govet`, `errcheck`, `staticcheck`, `dupl`, `bodyclose`, `exhaustive`, `errname`, and more. Zero linting errors are permitted.

Formatters enforce `gci`, `gofmt`, `gofumpt`, `goimports` with import ordering:

1. Standard library
2. Third-party packages
3. `github.com/gotenberg/gotenberg/v8`

Three groups separated by blank lines.

### Code Quality

- Errors are wrapped with context: `fmt.Errorf("description: %w", err)`. No swallowed errors.
- No business logic in `cmd/`.
- No panics in production code paths.
- Input is validated defensively.
- New features belong in the correct module (or justify a new one).

### Documentation

- Every exported function, type, constant, and variable has a GoDoc comment starting with its name.
- New packages include a `doc.go` file.
- `README.md` is not modified unless explicitly requested.

### Definition of Done

A change is ready to merge only when:

1. Code compiles: `go build ./...`
2. Code is formatted: `make fmt`
3. All linters pass: `make lint` and `make lint-prettier`
4. Integration tests pass: `make test-integration` (at minimum, the relevant `TAGS`)
5. Unit tests pass: `make test-unit`
6. All exported symbols and new packages have compliant GoDoc
7. Bruno collection is updated (if routes were added or modified)

---

## Bruno API Collection

A [Bruno](https://www.usebruno.com/) collection lives in `.bruno/` and mirrors every Gotenberg route. When adding or updating a route, update the collection to match.

### Structure

```
.bruno/
├── bruno.json                     # Collection config
├── collection.bru                 # Collection-level defaults (Gotenberg-Trace header)
├── environments/
│   ├── Local.bru                  # baseUrl: http://localhost:3000
│   └── Demo.bru                   # baseUrl: https://demo.gotenberg.dev
├── Health & Info/                  # GET routes
├── Chromium/Convert/               # POST routes grouped by module
├── Chromium/Screenshot/
├── LibreOffice/
└── PDF Engines/<Feature>/          # One folder per feature (Merge, Split, Rotate, …)
```

### `.bru` file format

```bru
meta {
  name: <Human-readable name>
  type: http
  seq: <order within folder>
}

post {
  url: {{baseUrl}}/forms/<path>
  body: multipartForm
  auth: none
}

body:multipart-form {
  files: @file(../../test/integration/testdata/<file>)
  <mandatoryField>: <value>
  ~<optionalField>: <value>
}

headers {
  ~Gotenberg-Output-Filename: <name>
  ~Gotenberg-Webhook-Url: http://localhost:8080/webhook
  ~Gotenberg-Webhook-Error-Url: http://localhost:8080/webhook/error
  ~Gotenberg-Webhook-Method: POST
  ~Gotenberg-Webhook-Error-Method: POST
  ~Gotenberg-Webhook-Extra-Http-Headers: {"X-Custom":"value"}
}
```

### Conventions

- **Mandatory fields** are listed without prefix; **optional fields** are prefixed with `~` (disabled by default in Bruno).
- **File references** use relative paths to `test/integration/testdata/`.
- **Webhook and output filename headers** are included on every POST route as optional (`~`).
- **One `.bru` file per request**. For routes with read/write variants (e.g., bookmarks, metadata), create separate files in the same folder.

### Checklist when adding/updating a route

1. Create or update the `.bru` file in the matching folder under `.bruno/`.
2. Include all form fields from the route handler — check `FormData*` calls in the route function.
3. For file upload fields (`files`, `watermark`, `stamp`, `embeds`), use `@file(...)` with a suitable test file.
4. Verify the URL path matches the route's `Path` field exactly.
5. If you add a new module folder, keep the naming consistent (e.g., `PDF Engines/Rotate/`).
