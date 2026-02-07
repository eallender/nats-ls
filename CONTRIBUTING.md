# Contributing to nats-ls

Thank you for your interest in contributing to nats-ls! We welcome contributions from the community.

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue with:
- A clear description of the problem
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Go version, NATS server version)
- Screenshots or logs if applicable

### Suggesting Features

Feature requests are welcome! Please open an issue describing:
- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** following our code style
   - Write clear, readable code
   - Add comments for complex logic
   - Follow Go conventions and idioms

3. **Test your changes**
   ```bash
   # Format code
   go fmt ./...

   # Run linter
   golangci-lint run

   # Run tests
   go test ./...

   # Or run all CI checks
   earthly +ci
   ```

4. **Commit your changes** with a clear commit message
   ```bash
   git commit -m "feat: add new feature"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `refactor:` for code refactoring
   - `test:` for adding tests
   - `chore:` for maintenance tasks

5. **Push to your fork** and submit a pull request
   ```bash
   git push origin feature/my-feature
   ```

6. **Wait for review** - maintainers will review your PR and may request changes

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/nats-ls.git
cd nats-ls

# Build
go build -o nls ./cmd/nls

# Run
./nls
```

## Code Quality

- Follow standard Go formatting (`go fmt`)
- Pass all linter checks (`golangci-lint run`)
- Write tests for new features
- Keep code simple and readable

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
