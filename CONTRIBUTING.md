# Contributing to cdd

Thanks for your interest in contributing to cdd, a terminal UI for jumping to
recently used Projects.

## Development setup

1. Install Go 1.26 or newer.
2. Clone the repository and fetch dependencies:

   ```sh
   git clone https://github.com/kryft-dev/cdd.git
   cd cdd
   go build ./...
   ```

3. Before pushing any change, run the full test suite:

   ```sh
   go test ./...
   ```

## Vocabulary

Before writing code or docs, read [`CONTEXT.md`](CONTEXT.md). It is the
glossary for this project: terms like Root, Kind, Project, Jump, Visit, Stale
Visit, History, Scan, Picker, and Wrapper have precise, agreed meanings.
Use those terms verbatim in code, comments, commit messages, and pull
requests instead of synonyms, so the codebase and its discussions share one
vocabulary.

## Workflow

* **Every change goes through a pull request.** Never push directly to
  `main`.
* **Commit in micro commits.** Each commit should be one logical change, and
  every commit message must follow the [Conventional Commits](https://www.conventionalcommits.org/)
  format (`feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`, ...).
* **Keep files small.** No file may exceed 300 lines. Split a file into
  smaller, focused files before it grows past that limit.
* **Test beside the code.** Tests live next to the code they cover, in an
  external test package (`package foo_test` for package `foo`), are
  table-driven where practical, and use only the standard library's
  `testing` package. Do not add testify or other assertion libraries.
* **Run the test suite before pushing.** Always run `go test ./...` (and
  ideally `go vet ./...`) before pushing a branch or opening a pull request.

## Reporting issues

Use GitHub Issues to report bugs or propose features. See the issue
templates for the information to include.

For security vulnerabilities, see [`SECURITY.md`](SECURITY.md) instead of
opening a public issue.
