# Contributing to Gotenberg

Thank you for your interest in contributing to Gotenberg! This guide will help you get started.

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

## Detailed Guidelines

The [`AGENTS.md`](AGENTS.md) files contain comprehensive guidelines used by both human contributors and AI-assisted tools:

| File                                                                   | What it covers                                                                                                            |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| [`AGENTS.md`](AGENTS.md)                                               | Core principles, mandatory workflow, project layout, coding patterns, module system, Makefile reference, review checklist |
| [`test/integration/AGENTS.md`](test/integration/AGENTS.md)             | Integration test framework (Godog/Gherkin), available tags, step reference, how to write new tests                        |
| [`.bruno/AGENTS.md`](.bruno/AGENTS.md)                                 | Bruno API collection structure, `.bru` file format, conventions, route update checklist                                   |
| [`pkg/modules/pdfengines/AGENTS.md`](pkg/modules/pdfengines/AGENTS.md) | How to add new PDF engine features (Makefile variable and flag)                                                           |
