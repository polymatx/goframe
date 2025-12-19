# Contributing to GoFrame

First off, thank you for considering contributing to GoFrame! It's people like you that make GoFrame such a great tool.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct: be respectful, inclusive, and constructive.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (code snippets, config files)
- **Describe the behavior you observed and what you expected**
- **Include Go version and OS information**

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description of the proposed feature**
- **Explain why this enhancement would be useful**
- **List any alternatives you've considered**

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. Ensure the test suite passes: `make test`
4. Make sure your code lints: `make lint`
5. Update documentation if needed
6. Issue the pull request

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/goframe.git
cd goframe

# Install dependencies
go mod download

# Run tests
make test

# Run linter
make lint

# Build
make build
```

## Style Guide

### Go Code

- Follow standard Go conventions and idioms
- Run `gofmt` and `goimports` on your code
- Use meaningful variable and function names
- Add comments for exported functions and types
- Handle errors properly - don't ignore them
- Use context for cancellation and timeouts

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or less
- Reference issues and pull requests liberally after the first line

Example:
```
Add rate limiting middleware

- Implement token bucket algorithm
- Add per-IP tracking with automatic cleanup
- Add configuration options

Fixes #123
```

### Documentation

- Update README.md for user-facing changes
- Update docs/DOCUMENTATION.md for detailed changes
- Add examples for new features
- Keep documentation concise and clear

## Project Structure

```
goframe/
├── cmd/goframe/      # CLI tool
├── pkg/              # Public packages
│   ├── app/          # Application core
│   ├── auth/         # Authentication
│   ├── middleware/   # HTTP middleware
│   ├── database/     # Database support
│   └── ...
├── examples/         # Example applications
├── docs/             # Documentation
└── build/            # Docker and deployment
```

## Testing

- Write table-driven tests when possible
- Use meaningful test names
- Test edge cases and error conditions
- Aim for good coverage on new code

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./pkg/auth/...
```

## Questions?

Feel free to open an issue with the "question" label or start a discussion.

Thank you for contributing! 🎉

