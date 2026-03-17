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

## Coding Patterns

- **Error handling:** Always wrap errors with context using `fmt.Errorf("description: %w", err)`. Never swallow errors silently.
- **Import ordering:** Enforced by `gci` — standard library, then third-party, then `github.com/gotenberg/gotenberg/v8`. Three groups separated by blank lines.
- **Mocks:** Comprehensive mock implementations for all major interfaces live in `pkg/gotenberg/mocks.go`. Use these for unit tests.
- **No business logic in `cmd/`:** The `cmd/gotenberg/` package is strictly for wiring and startup.

## Documentation

- Do not modify `README.md` unless explicitly asked.
- Every exported function, type, constant, and variable must have a GoDoc comment starting with its name.
- New packages must include a `doc.go` file with package-level documentation.
