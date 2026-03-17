# Contributing to Gotenberg

Thank you for your interest in contributing to Gotenberg! This guide will help you get started.

## Before You Start

Please read the [AGENTS.md](AGENTS.md) file — it describes the core principles, project layout, and development standards that all contributions must follow. Even though it is written for AI agents, the same rules apply to human contributors.

For deeper context on specific areas, see the personas in `.agents/`:

- **[DEVELOPER](.agents/DEVELOPER.md)** — Makefile workflow, module system, coding patterns.
- **[TESTER](.agents/TESTER.md)** — How to write integration and unit tests.
- **[REVIEWER](.agents/REVIEWER.md)** — What reviewers look for (useful to check before submitting).

## Getting Started

### Prerequisites

- Go (see version in `go.mod`)
- Docker
- Node.js (see version in `.node-version`) — for Prettier linting
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
```

To run only the integration tests relevant to your change:

```bash
make test-integration TAGS=health
make test-integration TAGS=chromium-convert-html
make test-integration TAGS="merge,split"
```

## Submitting a Pull Request

Before opening a PR, verify:

1. Code compiles: `make build`
2. Code is formatted: `make fmt` and `make prettify`
3. All linters pass: `make lint` and `make lint-prettier`
4. Integration tests pass: `make test-integration` (at minimum, the relevant tags)
5. Unit tests pass: `make test-unit`
6. All exported symbols and new packages have GoDoc comments

### Guidelines

- **Conventional Commits.** Commit messages must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification (e.g., `feat(chromium): add screenshot endpoint`, `fix(api): handle empty body`).
- **One thing per PR.** Keep features, bug fixes, and refactoring in separate PRs.
- **Backward compatibility matters.** Do not rename or remove existing CLI flags, environment variables, or API form fields without discussion.
- **Integration tests first.** When adding a feature or route, start by writing the Gherkin scenario in `test/integration/features/`.
- **No business logic in `cmd/`.** All logic belongs in `pkg/`.
