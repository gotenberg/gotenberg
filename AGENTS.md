# Operational Guidelines for Gotenberg

You are working on **Gotenberg**, a Docker-based API for converting documents to PDF. It is a widely used production dependency. Stability and backward compatibility are paramount. When in doubt about whether a change is breaking, flag it rather than assuming it's safe.

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

## Quick Reference

- Format before committing: `make fmt` (Go) and `make prettify` (non-Go)
- Lint before committing: `make lint && make lint-prettier`
- Commits must follow [Conventional Commits](https://www.conventionalcommits.org/) (e.g., `feat(chromium): add screenshot endpoint`)
- Run unit tests: `make test-unit`
- Run integration tests: `make build && make test-integration TAGS=<relevant-tag>`
- Never run `go` commands directly — use the Makefile.

## Codebase Navigation

- Start with `pkg/gotenberg/` for core interfaces and `pkg/modules/` for feature implementations.
- The integration test infrastructure in `test/integration/scenario/` is well-structured — read `scenario.go` and `containers.go` to understand the Gherkin step definitions before writing new tests.
- Mocks for all major interfaces are in `pkg/gotenberg/mocks.go` — use them for unit tests rather than creating new ones.
- Import ordering is enforced: standard library, third-party, then `github.com/gotenberg/gotenberg/v8` — separated by blank lines.
- When making changes, run only the relevant integration test tag rather than the full suite (40min timeout).

## Persona Selection (MANDATORY)

Before starting any task, you MUST read the appropriate persona file from `.agents/` based on what is being asked. This is not optional — the persona contains critical context you need.

| Task type                                                    | Persona to load                                | Trigger keywords / signals                                                        |
| ------------------------------------------------------------ | ---------------------------------------------- | --------------------------------------------------------------------------------- |
| Writing or modifying code (features, bug fixes, refactoring) | [`.agents/DEVELOPER.md`](.agents/DEVELOPER.md) | "add", "fix", "implement", "refactor", "change", "update", writing any `.go` file |
| Writing or updating tests                                    | [`.agents/TESTER.md`](.agents/TESTER.md)       | "test", "scenario", "coverage", `.feature` files, `_test.go` files                |
| Reviewing code or PRs                                        | [`.agents/REVIEWER.md`](.agents/REVIEWER.md)   | "review", "check", "audit", PR URLs, reviewing diffs                              |

If a task spans multiple concerns (e.g., implementing a feature AND writing tests), load ALL relevant personas.
