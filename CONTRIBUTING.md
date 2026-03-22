# Contributing to proto2pydantic

Thank you for your interest in contributing!

## Development Setup

1. **Prerequisites**: [Nix](https://nixos.org/download.html) (recommended) or Go 1.25+, buf, protoc
2. **Enter dev shell** (pick one):
   ```bash
   # Option A: direnv (auto-activates when you cd into the repo)
   direnv allow

   # Option B: manual
   nix develop
   ```
   Both install Go 1.25, buf, protoc, gh, golangci-lint, and pre-commit automatically.

3. **Pre-commit hooks** are installed automatically by the shell. They run `gofmt`, `go vet`, and `go mod tidy` on every commit. To run manually:
   ```bash
   pre-commit run --all-files
   ```

4. **Build**:
   ```bash
   go build -o protoc-gen-proto2pydantic .
   ```

5. **Test**:
   ```bash
   go test ./... -v
   ```

6. **Regenerate golden file** (after changing generator logic):
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

## Signed Commits

All commits to `main` must be signed. Set up Git commit signing:

```bash
# SSH signing (recommended)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true

# Or GPG signing
git config --global commit.gpgsign true
git config --global user.signingkey YOUR_GPG_KEY_ID
```

Make sure the signing key is added to your [GitHub account SSH keys](https://github.com/settings/keys) (type: "Signing Key").

## Code of Conduct

Be respectful and constructive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

## License

By contributing, you agree that your contributions will be licensed under the Apache-2.0 License.
