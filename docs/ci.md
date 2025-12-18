# CI/CD Documentation

## Overview

This project uses GitHub Actions for continuous integration and deployment. The CI pipeline runs on every push and pull request to ensure code quality and functionality.

## CI Pipeline

The CI workflow consists of three main jobs:

### 1. Lint
- Runs `golangci-lint` to check code quality
- Timeout: 5 minutes
- Configuration: `.golangci.yml`

### 2. Test
- Runs all unit tests with coverage
- Uses PostgreSQL 15 and RabbitMQ 3.12 services
- Executes database migrations
- Generates coverage report
- Uploads coverage to Codecov (optional)

### 3. Build
- Builds the API service binary
- Uploads artifact for download
- Retention: 7 days

## Running CI Checks Locally

### Prerequisites

Install required tools:
```bash
make install-lint
make install-tools
```

### Run All CI Checks

```bash
make ci
```

This will run:
1. Linter checks
2. All tests with coverage
3. Build verification

### Run Individual Checks

```bash
# Run linter only
make ci-lint

# Run tests only
make ci-test

# Build only
make ci-build
```

## Pre-commit Hooks (Optional)

Install pre-commit hooks to run checks before committing:

```bash
# Install pre-commit (requires Python)
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## GitHub Actions Workflow

**File:** `.github/workflows/ci.yml`

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Services:**
- PostgreSQL 15 (port 5432)
- RabbitMQ 3.12 (port 5672)

## Linter Configuration

**File:** `.golangci.yml`

**Enabled Linters:**
- gofmt, govet, staticcheck
- errcheck, gosimple, ineffassign
- unused, typecheck, misspell
- revive, gocyclo, dupl
- gosec, unconvert

**Settings:**
- Complexity threshold: 15
- Duplication threshold: 100 lines
- Timeout: 5 minutes

## Code Coverage

Current coverage: **16.2%** (target: 80%+)

**Breakdown:**
- Config package: 100%
- Logger package: 82.4%
- PostgreSQL/RabbitMQ: 0% (integration tests pending)

View coverage report:
```bash
make test-coverage
# Opens coverage.html in browser
```

## Branch Protection Rules (Recommended)

Configure in GitHub: `Settings > Branches > Add rule`

**For `main` branch:**
- ✅ Require a pull request before merging
- ✅ Require status checks to pass:
  - lint
  - test
  - build
- ✅ Require branches to be up to date
- ✅ Require conversation resolution

## Troubleshooting

### Linter Fails Locally But Passes in CI

Ensure you're using the same golangci-lint version:
```bash
golangci-lint --version
```

Update to latest:
```bash
make install-lint
```

### Tests Pass Locally But Fail in CI

Check environment variables and service availability:
```bash
# Verify PostgreSQL
docker ps | grep postgres

# Verify RabbitMQ
docker ps | grep rabbitmq
```

### Build Fails

Clear cache and rebuild:
```bash
make clean
go clean -modcache
make ci-build
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make ci` | Run all CI checks locally |
| `make ci-lint` | Run linter |
| `make ci-test` | Run tests with coverage |
| `make ci-build` | Build binary |
| `make install-lint` | Install golangci-lint |

## Next Steps

1. ✅ Set up branch protection rules
2. ✅ Configure Codecov integration
3. ⏳ Add security scanning (Dependabot)
4. ⏳ Add Docker image build/push workflow
5. ⏳ Add deployment workflow

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
