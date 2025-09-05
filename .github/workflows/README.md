# GitHub Actions Workflows

This directory contains the GitHub Actions workflows for the Cuju project. The workflows are split into separate pipelines for better organization, parallel execution, and easier maintenance.

## Workflow Overview

### 🔄 CI (`ci.yml`)
**Purpose**: Lightweight continuous integration with quick checks
- **Triggers**: Push to main/develop, PRs, manual dispatch
- **Jobs**:
  - `quick-check`: Fast formatting, vet, and build verification
  - `trigger-workflows`: Ensures dependent workflows are triggered

### 🏗️ Build (`build.yml`)
**Purpose**: Comprehensive build pipeline with multi-platform support
- **Triggers**: Push to main/develop, PRs, manual dispatch
- **Jobs**:
  - `build`: Single platform build with artifact upload
  - `build-matrix`: Multi-platform build matrix (Linux, Windows, macOS)

### 🧪 Test (`test.yml`)
**Purpose**: Comprehensive testing pipeline
- **Triggers**: Push to main/develop, PRs, manual dispatch
- **Jobs**:
  - `test`: Unit tests with race detection and coverage
  - `benchmark`: Performance benchmarks
  - `integration-test`: Integration tests

### 🔍 Lint (`lint.yml`)
**Purpose**: Code quality and security checks
- **Triggers**: Push to main/develop, PRs, manual dispatch
- **Jobs**:
  - `lint`: Code formatting, vet, and golangci-lint
  - `dependency-check`: Dependency vulnerability checks with Nancy

### 🚀 Release (Future)
**Purpose**: Automated releases and Docker image builds
- **Status**: Not yet implemented
- **Planned Features**:
  - Multi-platform binary builds
  - GitHub releases
  - Docker image builds

### 📦 Dependencies (Future)
**Purpose**: Automated dependency management
- **Status**: Not yet implemented
- **Planned Features**:
  - Weekly dependency updates
  - Security audits

## Workflow Benefits

### ✅ Parallel Execution
- Each workflow runs independently and in parallel
- Faster feedback on different aspects of the codebase
- Reduced overall CI time

### ✅ Focused Responsibilities
- Each workflow has a single, clear purpose
- Easier to debug and maintain
- Better separation of concerns

### ✅ Flexible Triggers
- Different workflows can be triggered independently
- Manual dispatch available for all workflows
- Scheduled workflows for maintenance tasks

### ✅ Resource Optimization
- Only necessary tools are installed per workflow
- Better caching strategies per workflow type
- Reduced resource usage

## Usage

### Manual Workflow Dispatch
All workflows support manual triggering:
1. Go to Actions tab in GitHub
2. Select the desired workflow
3. Click "Run workflow"
4. Choose branch and any required inputs

### Release Process
1. Create and push a git tag: `git tag v1.0.0 && git push origin v1.0.0`
2. The release workflow will automatically:
   - Build binaries for multiple platforms
   - Create a GitHub release
   - Build and push Docker images

### Dependency Updates
- Automatic weekly updates every Monday
- Manual trigger available
- Creates PRs for review before merging

## Configuration

### Environment Variables
- `GO_VERSION`: Set to '1.24' across all workflows
- `GITHUB_TOKEN`: Automatically provided by GitHub
- `DOCKER_USERNAME`/`DOCKER_PASSWORD`: Required for Docker releases

### Secrets Required
- `CODECOV_TOKEN`: Codecov token for coverage reporting (optional)
- `DOCKER_USERNAME`: Docker Hub username (for future releases)
- `DOCKER_PASSWORD`: Docker Hub password/token (for future releases)

## Monitoring

### Status Checks
Each workflow provides status checks that can be:
- Required for branch protection rules
- Used in PR merge requirements
- Monitored via GitHub API

### Artifacts
- Build artifacts are uploaded and available for download
- Coverage reports are generated and can be uploaded to services
- Security reports are archived for compliance

## Troubleshooting

### Common Issues
1. **Workflow not triggering**: Check branch names and trigger conditions
2. **Build failures**: Review Go version compatibility and dependencies
3. **Lint failures**: Run `make lint` locally to reproduce issues
4. **Test failures**: Check test environment and dependencies

### Local Testing
Before pushing, run locally:
```bash
make fmt      # Format code
make vet      # Run go vet
make lint     # Run golangci-lint
make test     # Run tests
make build    # Build binary
```
