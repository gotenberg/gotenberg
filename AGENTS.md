# Operational Guidelines for Gotenberg

As an AI agent working on the Gotenberg repository, you are expected to act with the diligence and architectural foresight of a Senior Go Engineer. Gotenberg is a widely used production dependency; stability and backward compatibility are paramount.

## 1. Core Philosophy & Stability

- **Backward Compatibility is Law:** This project creates a public API. Never modify existing flags, configuration environment variables, or API form fields unless explicitly instructed to perform a breaking change. If a change is breaking, it must be flagged immediately in the plan.
- **Defensive Programming:** Assume input data is malformed. Handle errors explicitly. Do not panic.
- **Atomic Commits:** Isolate refactoring from feature additions. A Pull Request should do one thing well.

## 2. Development Workflow & Tooling

You must rely strictly on the project's Makefile for build and verification tasks. Do not run `go` commands directly unless debugging a specific package requires it.

- **Formatting:** Run `make fmt` to format Go code before committing.
- **Linting:**
  - Run `make lint` to ensure Go code strictly adheres to the `.golangci.yml` configuration.
  - Run `make lint-prettier` to verify formatting for non-Go files (Markdown, YAML, etc.).
  - Zero linting errors are permitted.
- **Building:** Run `make build` to verify compilation and Docker image construction.

## 3. Architecture & Code Structure

- **Idiomatic Go:** Follow "Effective Go" principles.
- **Directory Separation:**
  - `cmd/`: Application entry points only. Contains wiring and startup logic. **No business logic is permitted here.**
  - `pkg/`: Core library code and modules. All business logic resides here.
- **Module System:** Gotenberg is modular (e.g., Chromium, LibreOffice). When adding features, determine if they belong to an existing module or require a new strict isolation.

## 4. Testing Standards

Gotenberg utilizes a split testing strategy. **Integration tests are the primary and preferred method for verifying features.**

- **Integration Tests (`make test-integration`):**
  - **First Priority:** Always start here when adding features or routes.
  - Gotenberg uses **Gherkin (Godog)** for end-to-end verification.
  - You **must** create or update the corresponding `.feature` file in `test/integration`.
  - These tests run within the Docker context; ensure environment consistency.
- **Unit Tests (`make test-unit`):**
  - Use table-driven tests for pure logic within `pkg/`.
  - Mock external dependencies (filesystem, network) where appropriate.

## 5. Documentation Requirements

- **No README Updates:** Do not modify the root `README.md` unless explicitly asked.
- **GoDoc is Mandatory:**
  - **New Packages:** If creating a new package, you must include a `doc.go` file containing the package-level documentation.
  - **Exported Symbols:** Every exported function, type, constant, and variable must have a proper GoDoc comment starting with its name.
  - **Quality:** Comments must be complete sentences explaining _what_ the symbol does and _how_ to use it.
  - **Example:**
    ```go
    // Convert transforms the input document to PDF using the Chromium engine.
    // It returns an error if the connection to the browser instance fails.
    func Convert(...) error
    ```

## 6. Definition of Done

A task is considered complete only when:

1.  The code compiles via `make build`.
2.  The code is formatted via `make fmt`.
3.  All linters pass via `make lint` and `make lint-prettier`.
4.  Integration scenarios pass via `make test-integration`.
5.  Unit tests pass via `make test-unit`.
6.  All exported symbols and new packages have compliant GoDoc.
