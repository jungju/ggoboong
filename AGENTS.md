# AGENTS.md

## Development Completion Rule

After finishing any development change in this project:

1. Run `go test ./...`.
2. Run `make build`.
3. Commit the completed change with a concise message.
4. Push the current branch to GitHub.
5. Create a GitHub Release for the next version tag with `gh release create`, including concise release notes for the change.

GitHub Pages deployment verification is not required for this repository.
