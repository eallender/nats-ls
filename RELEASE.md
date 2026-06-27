# Release Process

Releases are triggered by pushing a version tag from `main`.

## Branching Strategy

- `main` is the only long-lived branch.
- Feature/fix branches are created from `main` and merged back to `main` by pull request.
- The release workflow rejects tags whose target commit is not contained in `main`.

## Cutting a Release

1. Make sure `main` is at the commit you want to release.
2. Tag it with a `v`-prefixed semver version and push the tag:
   ```bash
   git checkout main
   git pull --ff-only origin main
   git tag v1.2.0
   git push origin v1.2.0
   ```
3. The [Release workflow](.github/workflows/release.yml) picks up the tag push and runs [GoReleaser](https://goreleaser.com/), which:
   - Verifies the tagged commit is on `main`
   - Runs formatting, vet, and tests
   - Builds binaries for all platforms, with the version baked in via `-ldflags`
   - Generates SHA256 checksums
   - Creates a GitHub release for the tag with grouped release notes (Features / Bug Fixes / Performance Improvements / Other Changes) generated from commits since the last tag, and the built binaries attached

Conventional commit prefixes (`feat:`, `fix:`, `perf:`) are used by GoReleaser to group entries in the release notes; everything else lands under "Other Changes."

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
