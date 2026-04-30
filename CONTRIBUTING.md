# Contributing to Gotenberg

Gotenberg is a Docker-based API for converting documents to PDF. Two rules override everything else:

- **Backward compatibility.** Never rename or remove CLI flags, environment variables, API form fields, or HTTP endpoints without discussion.
- **Defensive programming.** Assume input is malformed, handle errors explicitly, never panic.

## Toolchain

- Module: `github.com/gotenberg/gotenberg/v8`
- Go: see version in `go.mod`
- Docker
- Node.js (see `.node-version`), for Prettier linting
- [golangci-lint](https://golangci-lint.run/) v2+

## Before you start

For non-trivial changes, open an issue or a draft PR first. Describe what needs to change, the proposed solution (files to modify, interface changes, form fields), and which integration test tags are affected.

One thing per PR. Keep features, bug fixes, and refactoring in separate PRs.

When adding a feature or route, write the Gherkin scenario before the Go code, and plan to update the Bruno collection (`.bruno/`) if a route changes.

## Project layout

```
cmd/gotenberg/       -> Entry point only (wiring/startup). No business logic.
pkg/gotenberg/       -> Core module system, interfaces, utilities, mocks.
pkg/modules/         -> Feature modules (api, chromium, libreoffice, pdfengines, etc.).
pkg/standard/        -> Wires all standard modules together via imports.
test/integration/    -> Gherkin feature files + Go test infrastructure.
build/               -> Dockerfile, fonts, Chromium config.
.bruno/              -> Bruno API collection (mirrors every route).
```

Key interfaces live in `pkg/gotenberg/`: `Module`, `Provisioner`, `Validator`, `Debuggable`. Every module implements `Descriptor()` and self-registers via `init()`.

## Setup and Makefile

All build and verification tasks go through the Makefile. Do not run `go` commands directly unless debugging a specific package.

| Command                 | Purpose                                          | When to use                                                              |
| ----------------------- | ------------------------------------------------ | ------------------------------------------------------------------------ |
| `make build`            | Build the Gotenberg Docker image                 | Before integration tests or manual testing                               |
| `make run`              | Run a Gotenberg container via `docker compose`   | Manual testing. Flags configured via Makefile variables and compose.yaml |
| `make telemetry`        | Start an OpenTelemetry collector and OpenObserve | When testing telemetry locally                                           |
| `make down`             | Stop all compose containers                      | After manual testing                                                     |
| `make godoc`            | Serve GoDoc at `localhost:6060`                  | To verify documentation                                                  |
| `make fmt`              | Format Go code                                   | Before committing                                                        |
| `make lint`             | Lint Go code (zero errors permitted)             | Before committing                                                        |
| `make prettify`         | Format non-Go files (Markdown, YAML, JSON)       | Before committing                                                        |
| `make lint-prettier`    | Lint non-Go files                                | Before committing                                                        |
| `make test-unit`        | Run unit tests                                   | Before committing                                                        |
| `make test-integration` | Run all integration tests (40 min timeout)       | Before committing                                                        |

Run only the integration test tag(s) relevant to your change rather than the full suite:

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="merge,split"
```

## Code conventions

### Module system

Gotenberg uses a self-registering module architecture inspired by CaddyServer. Each module lives in `pkg/modules/<name>/`, implements at minimum `gotenberg.Module` (`Descriptor()`), and self-registers via `init()`. Wiring happens through `pkg/standard/`.

Determine if a feature belongs in an existing module before creating a new one. Only create a new module for a genuinely separate concern.

The `cmd/gotenberg/` package is strictly for wiring and startup. No business logic.

### Backward compatibility

CLI flags, environment variables, API form fields, HTTP endpoints, and default values that alter existing behavior must not change without discussion. Deprecate old names with `fs.MarkDeprecated()` and register both the old and new names side by side.

If a change violates backward compatibility, flag it as a breaking change in the PR description.

### Error handling

- Wrap every error with context: `fmt.Errorf("description: %w", err)`.
- Never swallow errors silently.
- Match errors with `errors.Is`, never `strings.Contains`.
- No panics in production code paths.
- Validate input defensively.

### Logging

Use `gotenberg.Logger(mod)` to get the module's slog logger during `Provision()`. All log calls must be context-aware: `logger.DebugContext(ctx, msg)`, `logger.InfoContext(ctx, msg)`, `logger.ErrorContext(ctx, msg)`. This propagates trace/span IDs into structured logs when OpenTelemetry is active.

### Telemetry

External tool calls (Chromium, LibreOffice, PDF engines, webhooks, downloads) must create OTEL spans with `trace.SpanKindClient` and `semconv.ServerAddress("toolname")`. Use `gotenberg.Tracer()` and `gotenberg.Meter()` for traces and metrics.

### Import ordering

Enforced by `gci`: standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.

## Documentation conventions

### Tone

- Short, declarative sentences. Say what it does, then stop.
- Lead with the action. "Validates font embedding", not "This function validates font embedding".
- Active voice. "Gotenberg checks the profile", not "The profile is checked by Gotenberg".
- No em dashes. Use a period, colon, or comma.
- No "we" hedging. "Don't...", not "We do not recommend...".

### Godoc

Every exported type and function has a Godoc comment starting with its identifier name:

```go
// Violation records a single rule violation with context.
type Violation struct { ... }

// ValidatePDFA audits the document against a PDF/A profile.
func ValidatePDFA(ctx context.Context, ...) ([]error, error)
```

Each package should have a `doc.go` with a `// Package foo ...` comment.

Reference identifiers with `[Name]` brackets for pkg.go.dev linking:

```go
// ValidatePDFA returns violations as []error where each element
// is a [Violation] value. See [Rule] for the structured fields.
```

### Code comments

- Explain _why_, not _what_.
- No numbered step comments (`// 1. Do X`, `// 2. Do Y`).
- No section dividers with numbers (`// --- 8. Foo ---`). Plain dividers are fine for major boundaries.
- No noise comments that restate the code (`// Check if err is nil`, `// Return results`).
- Reference spec clauses where relevant (`// Per ISO 32000-2, Table 116...`).
- Mark debt with `// TODO: [context]`.

## Testing

### Unit tests

Table-driven tests in `*_test.go` files. Use the comprehensive mock implementations in `pkg/gotenberg/mocks.go` rather than rolling new ones.

### Integration tests

Gherkin (BDD) via Godog with `testcontainers-go` for Docker orchestration. Feature files live in `test/integration/features/`; step definitions live in `test/integration/scenario/`. Read `scenario.go` and `containers.go` before writing new tests.

`make build` is required before running integration tests. The full suite has a 40-minute timeout, so run only the tag(s) relevant to your change.

## Pull requests

### Commits

[Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <description>`.

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`. The scope matches the module or area of the change (e.g., `chromium`, `pdfengines`, `api`).

Stage specific files. Never `git add -A` or `git add .`.

### Checklist

Before opening the PR, confirm:

- [ ] No backward-compatibility regression. See [Backward compatibility](#backward-compatibility).
- [ ] Code conventions met (error wrapping, logging, telemetry, import ordering, no panics, no business logic in `cmd/`). See [Code conventions](#code-conventions).
- [ ] Documentation conventions met (Godoc on every exported identifier, `doc.go` for new packages, tone). See [Documentation conventions](#documentation-conventions).
- [ ] `make fmt && make lint && make prettify && make lint-prettier` pass with zero warnings.
- [ ] `make test-unit` passes.
- [ ] Relevant `make test-integration TAGS=...` passes.
- [ ] Bruno collection updated if routes were added or modified.

## Further reading

- [`test/integration/README.md`](test/integration/README.md) — Gherkin step reference, available tags, writing new tests.
- [`.bruno/README.md`](.bruno/README.md) — `.bru` file format, conventions, route update checklist.
- [`pkg/modules/pdfengines/README.md`](pkg/modules/pdfengines/README.md) — adding new engine features (Makefile variable and flag).
