# Claude Code — Gotenberg

Read [AGENTS.md](AGENTS.md) first. It contains the core principles, project layout, and links to specialized personas you must load depending on the task.

## Quick Reference

- Format before committing: `make fmt` (Go) and `make prettify` (non-Go)
- Lint before committing: `make lint && make lint-prettier`
- Commits must follow [Conventional Commits](https://www.conventionalcommits.org/) (e.g., `feat(chromium): add screenshot endpoint`)
- Run unit tests: `make test-unit`
- Run integration tests: `make build && make test-integration TAGS=<relevant-tag>`
- Never run `go` commands directly — use the Makefile.

## Claude-Specific Guidance

- When exploring the codebase, start with `pkg/gotenberg/` for core interfaces and `pkg/modules/` for feature implementations.
- The integration test infrastructure in `test/integration/scenario/` is well-structured — read `scenario.go` and `containers.go` to understand the Gherkin step definitions before writing new tests.
- Mocks for all major interfaces are in `pkg/gotenberg/mocks.go` — use them for unit tests rather than creating new ones.
- Import ordering is enforced: standard library, third-party, then `github.com/gotenberg/gotenberg/v8` — separated by blank lines.
- When making changes, run only the relevant integration test tag rather than the full suite (40min timeout).
