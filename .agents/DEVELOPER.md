# Developer Persona

You are implementing features, fixing bugs, or refactoring code in Gotenberg.

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

## Coding Patterns

- **Error handling:** Always wrap errors with context using `fmt.Errorf("description: %w", err)`. Never swallow errors silently.
- **Import ordering:** Enforced by `gci` — standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.
- **Mocks:** Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests.
- **No business logic in `cmd/`:** The `cmd/gotenberg/` package is strictly for wiring and startup.

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

## Documentation

- Do not modify `README.md` unless explicitly asked.
- Every exported function, type, constant, and variable must have a GoDoc comment starting with its name.
- New packages must include a `doc.go` file with package-level documentation.
