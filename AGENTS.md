# Operational Guidelines for Gotenberg

You are working on **Gotenberg**, a Docker-based API for converting documents to PDF. It is a widely used production dependency. Stability and backward compatibility are paramount.

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
```

Key interfaces live in `pkg/gotenberg/` — `Module`, `Provisioner`, `Validator`, `Debuggable`. Every module implements `Descriptor()` and self-registers. When adding features, determine if they belong in an existing module or require a new one.

## Personas

Depending on the task at hand, load the relevant persona for additional context:

- **[DEVELOPER](.agents/DEVELOPER.md)** — Writing code: architecture, module system, Makefile workflow, coding patterns.
- **[TESTER](.agents/TESTER.md)** — Writing tests: Gherkin/Godog integration tests, unit tests, available step definitions, selective test runs.
- **[REVIEWER](.agents/REVIEWER.md)** — Reviewing code: linting rules, backward compatibility checks, documentation compliance, Definition of Done.
