# Contributing to proto2pydantic

Thank you for your interest in contributing!

## Development Setup

1. **Prerequisites**: [Nix](https://nixos.org/download.html) (recommended) or Go 1.25+, buf, protoc
2. **Enter dev shell**:
   ```bash
   nix develop
   ```
   This installs Go, buf, protoc, pre-commit hooks, and all dev tools automatically.

3. **Build**:
   ```bash
   go build -o protoc-gen-proto2pydantic .
   ```

4. **Test**:
   ```bash
   go test ./... -v
   ```

5. **Regenerate golden file** (after changing generator logic):
   ```bash
   cd testdata/proto && buf generate
   ```

## Pull Request Process

1. Fork the repo and create a feature branch
2. Make your changes
3. Run `go test ./...` and `go vet ./...`
4. Ensure `gofmt` is applied (pre-commit handles this automatically)
5. Update the golden file if generator output changes
6. Open a PR against `main`

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new features
- `fix:` bug fixes
- `chore:` maintenance
- `docs:` documentation
- `ci:` CI/CD changes
- `deps:` dependency updates

## Code of Conduct

Be respectful and constructive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

## License

By contributing, you agree that your contributions will be licensed under the Apache-2.0 License.
