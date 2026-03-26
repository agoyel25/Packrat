# Contributing to packrat

Thanks for your interest in contributing! Here's how to get started.

## Reporting issues

- Search [existing issues](https://github.com/agoyel25/Packrat/issues) before opening a new one.
- Include your OS/WSL version, the command you ran, and the full error output.

## Submitting a pull request

1. Fork the repo and create a branch from `main`.
2. Make your changes — keep commits focused and atomic.
3. Make sure the project builds cleanly: `go build ./...`
4. Open a PR with a clear description of what you changed and why.

## Development setup

```bash
git clone https://github.com/agoyel25/Packrat.git
cd Packrat
go build -o packrat
./packrat --help
```

## Guidelines

- Follow standard Go conventions (`gofmt`, idiomatic error handling).
- Keep new dependencies minimal — this tool is intentionally lean.
- If you're adding a new backup category, update `internal/categories/categories.go` and the README table.

## Questions?

Open a [GitHub issue](https://github.com/agoyel25/Packrat/issues) and tag it `question`.
