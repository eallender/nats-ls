# Release Process

This project uses [semantic-release](https://semantic-release.gitbook.io/) to automate versioning and releases based on conventional commits.

## Branching Strategy

- **`dev`**: Development branch - CI runs tests and builds, no releases created
- **`main`**: Production branch - creates stable releases (e.g., `1.2.0`)

## Workflow

1. **Feature Development**
   - Create feature branches from `dev`
   - Commit using conventional commit format (see below)
   - Open PR to merge into `dev`

2. **Development (dev branch)**
   - When PRs are merged to `dev`:
     - CI runs tests and builds
     - No releases are created
     - Commits accumulate for the next release

3. **Stable Release (main branch)**
   - When `dev` is merged to `main`, semantic-release automatically:
     - Analyzes all commits since the last release
     - Determines version based on the highest severity change
     - Creates a stable version (e.g., `1.2.0`)
     - Builds binaries for all platforms
     - Creates a GitHub release with artifacts
     - Generates and updates CHANGELOG.md
     - Commits the changelog back to the repository

## Conventional Commits

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Commit Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types and Version Bumps

| Type | Description | Version Bump | Example |
|------|-------------|--------------|---------|
| `feat` | New feature | **Minor** (0.x.0) | `feat: add subject filtering` |
| `fix` | Bug fix | **Patch** (0.0.x) | `fix: resolve connection timeout` |
| `perf` | Performance improvement | **Patch** (0.0.x) | `perf: optimize message parsing` |
| `BREAKING CHANGE` | Breaking change | **Major** (x.0.0) | See below |

### Types that DON'T trigger releases

- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `build`: Build system changes

### Breaking Changes

To trigger a major version bump, include `BREAKING CHANGE:` in the commit footer:

```
feat: redesign configuration format

BREAKING CHANGE: Configuration file format has changed from YAML to TOML.
Users must migrate their existing config files.
```

Or use `!` after the type:

```
feat!: redesign configuration format
```

### Examples

#### Minor version bump (new feature)
```
feat(tui): add dark mode support

Added configurable dark mode theme for better visibility
in different lighting conditions.
```

#### Patch version bump (bug fix)
```
fix(connection): handle reconnection edge case

Fixed issue where reconnection would fail if the initial
connection was interrupted during handshake.
```

#### Major version bump (breaking change)
```
feat!: change CLI flag names for consistency

BREAKING CHANGE: Renamed --nats-server to --server and
--nats-url to --url for consistency with other CLI tools.
```

## Manual Release

While semantic-release automates the process, you can manually trigger a release by:

1. Ensuring you're on the correct branch (`dev` or `main`)
2. Pushing commits with conventional commit messages
3. The GitHub Action will automatically run and create the release

## Release Assets

Each release includes:
- `nls-linux-amd64` - Linux binary (AMD64)
- `nls-linux-arm64` - Linux binary (ARM64)
- `nls-darwin-amd64` - macOS binary (AMD64/Intel)
- `nls-darwin-arm64` - macOS binary (ARM64/Apple Silicon)
- `checksums.txt` - SHA256 checksums for all binaries

## Checking Release Status

- View release status: Check the [Actions tab](../../actions/workflows/release.yml)
- View releases: Check the [Releases page](../../releases)
- View changelog: See [CHANGELOG.md](./CHANGELOG.md)
