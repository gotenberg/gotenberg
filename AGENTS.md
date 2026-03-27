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

- **Integration tests** (primary): Gherkin scenarios in `test/integration/features/`. See [`test/integration/AGENTS.md`](test/integration/AGENTS.md) for the full reference.
- **Unit tests** (when applicable): Table-driven tests in `*_test.go` files using mocks from `pkg/gotenberg/mocks.go`.

### Step 4 — Review

Self-review the implementation against the [Review Checklist](#review-checklist). Fix any issues found before presenting the result to the user.

### Step 5 — Commit

Present the review to the user and **wait for explicit approval**. Do NOT commit until the user confirms. Once approved, create a commit following the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>
```

Common types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`, `build`. The scope should match the module or area of the change (e.g., `chromium`, `pdfengines`, `api`).

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
- Telemetry infrastructure lives in `pkg/gotenberg/telemetry.go` (global Logger, Tracer, Meter) and `pkg/gotenberg/internal/` (log handlers, OTEL SDK init). HTTP semantic conventions are in `pkg/gotenberg/semconv/`.

---

## Makefile — the Only Build Interface

All build and verification tasks go through the Makefile. Do not run `go` commands directly unless debugging a specific package.

| Command                 | Purpose                                                       | When to use                                                                  |
| ----------------------- | ------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `make build`            | Build the Docker image                                        | Before integration tests, or to verify compilation                           |
| `make run`              | Run Gotenberg container via `docker compose`                  | Manual testing. Flags are configured via Makefile variables and compose.yaml |
| `make telemetry`        | Start OpenTelemetry collector and OpenObserve                 | When testing telemetry locally                                               |
| `make down`             | Stop all compose containers                                   | After manual testing                                                         |
| `make fmt`              | Format Go code (`go fix`, `golangci-lint fmt`, `go mod tidy`) | Before every commit                                                          |
| `make lint`             | Lint Go code (strict `.golangci.yml` config)                  | Before every commit. Zero errors permitted                                   |
| `make lint-prettier`    | Lint non-Go files (Markdown, YAML, etc.) with Prettier        | Before every commit                                                          |
| `make prettify`         | Format non-Go files (Markdown, YAML, etc.) with Prettier      | Before every commit                                                          |
| `make test-unit`        | Run unit tests (`go test -race ./...`)                        | After code changes to `pkg/`                                                 |
| `make test-integration` | Run integration tests (Gherkin/Godog, 40min timeout)          | After any feature or route change                                            |
| `make godoc`            | Serve GoDoc at `localhost:6060`                               | To verify documentation                                                      |

## Module System

Gotenberg uses a self-registering module architecture inspired by CaddyServer. Each module:

- Lives in `pkg/modules/<name>/`
- Implements the `gotenberg.Module` interface (at minimum `Descriptor()`)
- May also implement `gotenberg.Provisioner`, `gotenberg.Validator`, or `gotenberg.Debuggable`
- Self-registers via `init()` and is wired through `pkg/standard/`

When adding a feature, first determine if it belongs in an existing module. Only create a new module if the feature represents a genuinely separate concern.

## Coding Patterns

- **Error handling:** Always wrap errors with context using `fmt.Errorf("description: %w", err)`. Never swallow errors silently.
- **Import ordering:** Enforced by `gci` — standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.
- **Mocks:** Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests.
- **Logging:** Use `gotenberg.Logger(mod)` to get the module's slog logger during `Provision()`. All log calls must be context-aware: `logger.DebugContext(ctx, msg)`, `logger.InfoContext(ctx, msg)`, `logger.ErrorContext(ctx, msg)`. This propagates trace/span IDs into structured logs when OpenTelemetry is active.
- **Telemetry:** External tool calls (Chromium, LibreOffice, PDF engines, webhooks, downloads) must create OTEL spans with `trace.SpanKindClient` and `semconv.ServerAddress("toolname")`. Use `gotenberg.Tracer()` and `gotenberg.Meter()` for traces and metrics respectively.
- **No business logic in `cmd/`:** The `cmd/gotenberg/` package is strictly for wiring and startup.

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

## Scoped Guidelines

Detailed guidelines for specific areas of the codebase live in their own `AGENTS.md` files:

- [`test/integration/AGENTS.md`](test/integration/AGENTS.md) — Integration test framework, Gherkin step reference, available tags, and how to write new tests.
- [`.bruno/AGENTS.md`](.bruno/AGENTS.md) — Bruno API collection structure, `.bru` file format, conventions, and route update checklist.
- [`pkg/modules/pdfengines/AGENTS.md`](pkg/modules/pdfengines/AGENTS.md) — How to add new PDF engine features (Makefile variable and flag).
