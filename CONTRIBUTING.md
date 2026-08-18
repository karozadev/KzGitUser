# Contributing to KzGitUser

Thanks for considering a contribution! This document explains how to get set up and how to propose a change.

## Local setup

Requirements: [Go](https://go.dev/dl/) (see the version in [go.mod](go.mod)) and `git`.

```bash
git clone git@github.com:karozadev/KzGitUser.git
cd KzGitUser
go build -o kzgit .
./kzgit whoami
```

## Running tests

```bash
go test ./...
```

With coverage:

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

Tests are isolated from your personal Git configuration (they redirect `HOME`, `GIT_CONFIG_GLOBAL` and `GIT_CONFIG_SYSTEM` to temporary directories), so they're safe to run repeatedly and won't touch your real `~/.gitconfig` or KzGitUser config.

## Running linters

We use [golangci-lint](https://golangci-lint.run/):

```bash
golangci-lint run ./...
```

Format code with:

```bash
gofmt -l .   # list files that need formatting
gofmt -w .   # apply formatting
```

## Commit conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<optional scope>): <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`. Example:

```text
feat(check): support multiple --domain flags
```

## Git workflow (GitFlow)

This project follows a lightweight GitFlow:

- `main` — always releasable; tagged releases (`vX.Y.Z`) are cut from here.
- `feature/*` — new features and non-trivial changes, branched from `main`, merged back via pull request.
- `fix/*` — bug fixes, same flow as feature branches.

1. Branch from `main`: `git checkout -b feature/my-change`.
2. Make your changes with clear, focused commits.
3. Ensure `go build ./...`, `go test ./...`, and `golangci-lint run ./...` all pass.
4. Push your branch and open a pull request against `main`.

## Opening a pull request

- Keep PRs focused on a single change; small PRs are easier to review.
- Describe *why* the change is needed, not just what changed.
- Add or update tests for any behavior change.
- Update `README.md` if you change user-facing behavior (commands, flags, output format).
- Make sure CI is green before requesting review.

## Design principles

KzGitUser is intentionally a small, fast CLI. When contributing:

- Prefer using `git` itself (via `internal/git`) over reimplementing Git's config resolution.
- Keep `internal/git` and `internal/config` free of CLI/Cobra concerns — they should be usable and testable on their own.
- Avoid adding dependencies unless they clearly pay for themselves.
- Don't add speculative features or configuration options that aren't needed yet.
