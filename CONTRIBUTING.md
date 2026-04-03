# Contributing to Gotenberg

**Gotenberg** is a Docker-based API for converting documents to PDF. It is a widely used production dependency. Stability and backward compatibility are paramount. When in doubt about whether a change is breaking, flag it rather than assuming it's safe.

## Getting Started

### Prerequisites

- Go (see version in `go.mod`)
- Docker
- Node.js (see version in `.node-version`), for Prettier linting
- [golangci-lint](https://golangci-lint.run/) v2+

### Build and Run

```bash
make build # Build the Docker image
make run   # Run a local Gotenberg container
```

### Development Loop

```bash
# Write your code, then:
make fmt              # Format Go code
make prettify         # Format non-Go files (Markdown, YAML, etc.)
make lint             # Lint Go code (zero errors permitted)
make lint-prettier    # Lint non-Go files
make test-unit        # Run unit tests
make build            # Build the Docker image (required before integration tests)
make test-integration # Run all integration tests
make telemetry        # Start OpenTelemetry collector and OpenObserve
make down             # Stop all compose containers
```

To run only the integration tests relevant to your change:

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="merge,split"
```

## Submitting a Pull Request

For non-trivial changes, outline your approach before writing code. Open an issue or draft PR describing:

- What needs to change and why.
- The proposed solution, with enough detail to implement (files to modify, interface changes, form fields, etc.).
- Which integration test tags will be affected and what new scenarios are needed.

Before opening (or marking ready) a PR, verify:

1. Code compiles: `make build`
2. Code is formatted: `make fmt` and `make prettify`
3. All linters pass: `make lint` and `make lint-prettier`
4. Integration tests pass: `make test-integration` (at minimum, the relevant tags)
5. Unit tests pass: `make test-unit`
6. All exported symbols and new packages have GoDoc comments
7. Bruno collection is updated (if routes were added or modified)

Review your changes against the [Review Checklist](#review-checklist) before submitting.

### Guidelines

- **One thing per PR.** Keep features, bug fixes, and refactoring in separate PRs.
- **Backward compatibility matters.** Do not rename or remove existing CLI flags, environment variables, or API form fields without discussion.
- **Integration tests first.** When adding a feature or route, start by writing the Gherkin scenario in `test/integration/features/`. See [`test/integration/README.md`](test/integration/README.md) for the full reference.
- **Unit tests** when applicable: table-driven tests in `*_test.go` files using mocks from `pkg/gotenberg/mocks.go`.

### Commit Conventions

If committing, follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`. The scope should match the module or area of the change (e.g., `chromium`, `pdfengines`, `api`).

Stage only the files related to the change. Do not use `git add -A` or `git add .`.

---

## Core Principles

- **Backward compatibility is law.** See the [Review Checklist](#review-checklist) for the full list of what must not change.
- **Defensive programming.** Assume input is malformed. Handle errors explicitly. Never panic.
- **Atomic commits.** One feature or fix per PR. Isolate refactoring from feature work.
- **Idiomatic Go.** Follow "Effective Go" principles. All exported symbols must have GoDoc comments starting with their name.

## Project Layout and Navigation

```
cmd/gotenberg/       → Entry point only (wiring/startup). No business logic.
pkg/gotenberg/       → Core module system, interfaces, utilities, mocks.
pkg/modules/         → Feature modules (api, chromium, libreoffice, pdfengines, etc.).
pkg/standard/        → Wires all standard modules together via imports.
test/integration/    → Gherkin feature files + Go test infrastructure.
build/               → Dockerfile, fonts, Chromium config.
.bruno/              → Bruno API collection (mirrors every route).
```

Key interfaces live in `pkg/gotenberg/`: `Module`, `Provisioner`, `Validator`, `Debuggable`. Every module implements `Descriptor()` and self-registers. When adding features, determine if they belong in an existing module or require a new one.

- The integration test infrastructure in `test/integration/scenario/` is well-structured. Read `scenario.go` and `containers.go` to understand the Gherkin step definitions before writing new tests.
- Mocks for all major interfaces are in `pkg/gotenberg/mocks.go`. Use them for unit tests rather than creating new ones.
- When making changes, run only the relevant integration test tag rather than the full suite (40min timeout).
- Telemetry infrastructure lives in `pkg/gotenberg/telemetry.go` (global Logger, Tracer, Meter) and `pkg/gotenberg/internal/` (log handlers, OTEL SDK init). HTTP semantic conventions are in `pkg/gotenberg/semconv/`.

## Makefile: the Only Build Interface

All build and verification tasks go through the Makefile. Do not run `go` commands directly unless debugging a specific package. The [Development Loop](#development-loop) covers the commands used during daily work. Additional commands:

| Command          | Purpose                                       | When to use                                                                  |
| ---------------- | --------------------------------------------- | ---------------------------------------------------------------------------- |
| `make run`       | Run Gotenberg container via `docker compose`  | Manual testing. Flags are configured via Makefile variables and compose.yaml |
| `make telemetry` | Start OpenTelemetry collector and OpenObserve | When testing telemetry locally                                               |
| `make down`      | Stop all compose containers                   | After manual testing                                                         |
| `make godoc`     | Serve GoDoc at `localhost:6060`               | To verify documentation                                                      |

## Module System

Gotenberg uses a self-registering module architecture inspired by CaddyServer. Each module:

- Lives in `pkg/modules/<name>/`
- Implements the `gotenberg.Module` interface (at minimum `Descriptor()`)
- May also implement `gotenberg.Provisioner`, `gotenberg.Validator`, or `gotenberg.Debuggable`
- Self-registers via `init()` and is wired through `pkg/standard/`

When adding a feature, first determine if it belongs in an existing module. Only create a new module if the feature represents a genuinely separate concern.

## Coding Patterns

- **Error handling:** Always wrap errors with context using `fmt.Errorf("description: %w", err)`. Never swallow errors silently.
- **Import ordering:** Enforced by `gci`: standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.
- **Mocks:** Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests.
- **Logging:** Use `gotenberg.Logger(mod)` to get the module's slog logger during `Provision()`. All log calls must be context-aware: `logger.DebugContext(ctx, msg)`, `logger.InfoContext(ctx, msg)`, `logger.ErrorContext(ctx, msg)`. This propagates trace/span IDs into structured logs when OpenTelemetry is active.
- **Telemetry:** External tool calls (Chromium, LibreOffice, PDF engines, webhooks, downloads) must create OTEL spans with `trace.SpanKindClient` and `semconv.ServerAddress("toolname")`. Use `gotenberg.Tracer()` and `gotenberg.Meter()` for traces and metrics respectively.
- **No business logic in `cmd/`:** The `cmd/gotenberg/` package is strictly for wiring and startup.

## Documentation

### Writing Style

- **Short, declarative sentences.** Say what it does, then stop.
- **Lead with the action.** "Validates font embedding" not "This function validates font embedding".
- **Active voice.** "Gotenberg checks the profile" not "The profile is checked by Gotenberg".
- **No em dashes.** Use a period, colon, or comma instead.
- **No "we" hedging.** "Don't..." not "We do not recommend...".

### Godoc

All exported types and functions require Godoc comments. Start with the identifier name:

```go
// Violation records a single rule violation with context.
type Violation struct { ... }

// ValidatePDFA audits the document against a PDF/A profile.
func ValidatePDFA(ctx context.Context, ...) ([]error, error)
```

Each package should have a `doc.go` with a `// Package foo ...` comment.

Reference other identifiers with square brackets so pkg.go.dev renders them as links:

```go
// ValidatePDFA returns violations as []error where each element is a
// [Violation] value. See [Rule] for the structured rule fields.
// The document must be opened via [pdf.Open] with an [io.ReaderAt].
```

This works for same-package identifiers (`[Violation]`), other packages (`[io.Reader]`), and methods (`[Reader.Open]`).

### Code Comments

- Explain _why_, not _what_. The code shows what; the comment explains the non-obvious reasoning.
- No numbered step comments (`// 1. Do X`, `// 2. Do Y`).
- No section dividers with numbers (`// --- 8. Foo ---`). Plain dividers are fine for major boundaries (`// --- VeraPDF ---`).
- No noise comments that restate the code (`// Check if err is nil`, `// Return results`).
- Reference spec clauses where relevant (`// Per ISO 32000-2, Table 116...`).
- Mark technical debt with `// TODO: [context]`.

---

## Review Checklist

### Backward Compatibility

- [ ] No existing CLI flags renamed or removed
- [ ] No existing environment variables renamed or removed
- [ ] No existing API form fields renamed or removed
- [ ] No existing HTTP endpoints changed or removed
- [ ] No changes to default values that alter existing behavior
- [ ] Deprecated flags have both old and new names registered, with `fs.MarkDeprecated()`

If any of these are violated, the change **must** be flagged as a breaking change.

### Linting Standards

The `.golangci.yml` enforces strict rules including: `gosec`, `govet`, `errcheck`, `staticcheck`, `dupl`, `bodyclose`, `exhaustive`, `errname`, `sloglint`, `gocritic`, and more. Zero linting errors are permitted.

Formatters enforce `gci`, `gofmt`, `gofumpt`, `goimports` (see import ordering in [Coding Patterns](#coding-patterns)).

### Code Quality

- Errors are wrapped with context: `fmt.Errorf("description: %w", err)`. No swallowed errors.
- No business logic in `cmd/`.
- No panics in production code paths.
- Input is validated defensively.
- New features belong in the correct module (or justify a new one).

### Documentation

- Every exported function, type, constant, and variable has a Godoc comment starting with its name (see [Godoc](#godoc)).
- New packages include a `doc.go` file.
- `README.md` is not modified unless explicitly requested.
- All documentation follows the [Writing Style](#writing-style) and [Code Comments](#code-comments) guidelines.

---

## Scoped Guidelines

Some areas of the codebase have their own README with detailed instructions:

| Area              | README                                                                 | Covers                                                    |
| ----------------- | ---------------------------------------------------------------------- | --------------------------------------------------------- |
| Integration tests | [`test/integration/README.md`](test/integration/README.md)             | Gherkin step reference, available tags, writing new tests |
| Bruno collection  | [`.bruno/README.md`](.bruno/README.md)                                 | `.bru` file format, conventions, route update checklist   |
| PDF engines       | [`pkg/modules/pdfengines/README.md`](pkg/modules/pdfengines/README.md) | Adding new engine features (Makefile variable and flag)   |
