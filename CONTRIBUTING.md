# Contributing to Gotenberg

Gotenberg is a Docker-based API for converting documents to PDF. Two rules override everything else: **backward compatibility** (never rename or remove CLI flags, environment variables, API form fields, or HTTP endpoints without discussion) and **defensive programming** (assume input is malformed, handle errors explicitly, never panic).

- Module: `github.com/gotenberg/gotenberg/v8`
- Go: see version in `go.mod`
- Docker
- Node.js (see `.node-version`), for Prettier linting
- [golangci-lint](https://golangci-lint.run/) v2+

## Quick start

```bash
make build            # Build the Docker image
make run              # Run a local Gotenberg container
make fmt              # Format Go code
make prettify         # Format non-Go files (Markdown, YAML, etc.)
make lint             # Lint Go code (zero errors permitted)
make lint-prettier    # Lint non-Go files
make test-unit        # Run unit tests
make build            # Required before integration tests
make test-integration # Run all integration tests
make telemetry        # Start OpenTelemetry collector and OpenObserve
make down             # Stop all compose containers
```

Run only the integration tests relevant to your change:

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="merge,split"
```

All build and verification tasks go through the Makefile. Do not run `go` commands directly unless debugging a specific package.

| Command          | Purpose                                       | When to use                                                              |
| ---------------- | --------------------------------------------- | ------------------------------------------------------------------------ |
| `make run`       | Run Gotenberg container via `docker compose`  | Manual testing. Flags configured via Makefile variables and compose.yaml |
| `make telemetry` | Start OpenTelemetry collector and OpenObserve | When testing telemetry locally                                           |
| `make down`      | Stop all compose containers                   | After manual testing                                                     |
| `make godoc`     | Serve GoDoc at `localhost:6060`               | To verify documentation                                                  |

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

Key interfaces live in `pkg/gotenberg/`: `Module`, `Provisioner`, `Validator`, `Debuggable`. Every module implements `Descriptor()` and self-registers.

## Coding rules

### Module system

Gotenberg uses a self-registering module architecture inspired by CaddyServer. Each module lives in `pkg/modules/<name>/`, implements at minimum `gotenberg.Module` (`Descriptor()`), and self-registers via `init()`. Wiring happens through `pkg/standard/`.

Determine if a feature belongs in an existing module before creating a new one. Only create a new module for a genuinely separate concern.

### Error handling

- Wrap every error with context: `fmt.Errorf("description: %w", err)`.
- Never swallow errors silently.
- Match errors with `errors.Is`, never `strings.Contains`.
- No panics in production code paths.
- Input is validated defensively.

### Import ordering

Enforced by `gci`: standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.

### Logging

Use `gotenberg.Logger(mod)` to get the module's slog logger during `Provision()`. All log calls must be context-aware: `logger.DebugContext(ctx, msg)`, `logger.InfoContext(ctx, msg)`, `logger.ErrorContext(ctx, msg)`. This propagates trace/span IDs into structured logs when OpenTelemetry is active.

### Telemetry

External tool calls (Chromium, LibreOffice, PDF engines, webhooks, downloads) must create OTEL spans with `trace.SpanKindClient` and `semconv.ServerAddress("toolname")`. Use `gotenberg.Tracer()` and `gotenberg.Meter()` for traces and metrics.

### No business logic in `cmd/`

The `cmd/gotenberg/` package is strictly for wiring and startup.

### Mocks

Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests rather than creating new ones.

## Documentation rules

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

### Formatting non-Go files

Run `make prettify && make lint-prettier` for YAML, Markdown, and JSON.

## Testing

Integration tests use Gherkin (BDD) via Godog with `testcontainers-go` for Docker orchestration. Feature files live in `test/integration/features/`. Step definitions live in `test/integration/scenario/`. Read `scenario.go` and `containers.go` to understand step definitions before writing new tests.

Unit tests: table-driven tests in `*_test.go` files using mocks from `pkg/gotenberg/mocks.go`.

Run only the relevant integration test tag rather than the full suite (40min timeout).

See:

- [`test/integration/README.md`](test/integration/README.md): Gherkin step reference, available tags, writing new tests.
- [`.bruno/README.md`](.bruno/README.md): `.bru` file format, conventions, route update checklist.
- [`pkg/modules/pdfengines/README.md`](pkg/modules/pdfengines/README.md): adding new engine features (Makefile variable and flag).

## Pull requests

Plan non-trivial changes before coding. Open an issue or draft PR describing what needs to change, the proposed solution (files to modify, interface changes, form fields), and which integration test tags are affected.

### Guidelines

- One thing per PR. Keep features, bug fixes, and refactoring separate.
- Backward compatibility matters. Do not rename or remove existing CLI flags, environment variables, or API form fields without discussion.
- Integration tests first. When adding a feature or route, start by writing the Gherkin scenario.
- Bruno collection must be updated if routes were added or modified.

### Checklist

- [ ] No existing CLI flags renamed or removed
- [ ] No existing environment variables renamed or removed
- [ ] No existing API form fields renamed or removed
- [ ] No existing HTTP endpoints changed or removed
- [ ] No changes to default values that alter existing behavior
- [ ] Deprecated flags have both old and new names registered, with `fs.MarkDeprecated()`
- [ ] Errors wrapped with context: `fmt.Errorf("description: %w", err)`
- [ ] No business logic in `cmd/`
- [ ] No panics in production code paths
- [ ] New features belong in the correct module (or justify a new one)
- [ ] Linting: `make fmt && make lint && make prettify && make lint-prettier` passes with zero warnings
- [ ] Every exported function, type, constant, and variable has a Godoc comment starting with its name
- [ ] New packages include a `doc.go` file
- [ ] Integration tests pass: `make test-integration` (at minimum, the relevant tags)
- [ ] Unit tests pass: `make test-unit`
- [ ] Bruno collection updated (if routes were added or modified)

If any backward compatibility item is violated, the change **must** be flagged as a breaking change.

### Commits

[Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <description>`.

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`. The scope should match the module or area of the change (e.g., `chromium`, `pdfengines`, `api`).

Stage specific files. Never `git add -A` or `git add .`. Do not push unless asked.
