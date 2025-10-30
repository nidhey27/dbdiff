# Release Guide

This document explains how to create releases for the dbdiff project.

## Automated Releases (Recommended)

The project uses GitHub Actions to automatically build and publish releases.

### Steps to Create a Release

1. **Ensure all changes are committed and pushed to main:**
   ```bash
   git add .
   git commit -m "Prepare for release v1.0.0"
   git push origin main
   ```

2. **Create and push a version tag:**
   ```bash
   # Create an annotated tag
   git tag -a v1.0.0 -m "Release version 1.0.0"
   
   # Push the tag to GitHub
   git push origin v1.0.0
   ```

3. **GitHub Actions will automatically:**
   - Detect the new tag
   - Build binaries for all platforms:
     - Linux (AMD64, ARM64)
     - macOS (AMD64/Intel, ARM64/Apple Silicon)
     - Windows (AMD64, ARM64)
   - Generate SHA256 checksums
   - Create a GitHub Release
   - Attach all binaries and checksums to the release
   - Add installation instructions to release notes

4. **Verify the release:**
   - Go to `https://github.com/YOUR_USERNAME/dbdiff/releases`
   - Check that all binaries are attached
   - Test download and installation instructions

## Manual Local Builds

If you need to build binaries locally without creating a release:

### Build All Platforms

```bash
# Build all platform binaries
make build-all

# Binaries will be in build/ directory
ls -lh build/
```

### Build with Version

```bash
# Build release with specific version
make release VERSION=1.0.0

# This creates binaries and checksums in build/
```

### Build Single Platform

```bash
# Build for current platform only
make build

# Or specify platform manually
GOOS=linux GOARCH=amd64 go build -o dbdiff-linux-amd64 main.go
```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v2.0.0): Incompatible API changes
- **MINOR** version (v1.1.0): New functionality, backwards compatible
- **PATCH** version (v1.0.1): Bug fixes, backwards compatible

Examples:
- `v1.0.0` - First stable release
- `v1.1.0` - Added MySQL support
- `v1.1.1` - Fixed connection bug
- `v2.0.0` - Changed CLI flags (breaking change)

## Pre-releases

For beta or release candidate versions:

```bash
git tag -a v1.0.0-beta.1 -m "Beta release 1.0.0-beta.1"
git push origin v1.0.0-beta.1
```

Mark as pre-release in GitHub UI after automatic creation.

## Supported Platforms

The release workflow builds for:

| OS      | Architecture | Binary Name                  |
|---------|--------------|------------------------------|
| Linux   | AMD64        | dbdiff-linux-amd64           |
| Linux   | ARM64        | dbdiff-linux-arm64           |
| macOS   | AMD64        | dbdiff-darwin-amd64          |
| macOS   | ARM64        | dbdiff-darwin-arm64          |
| Windows | AMD64        | dbdiff-windows-amd64.exe     |
| Windows | ARM64        | dbdiff-windows-arm64.exe     |

## Checksums

SHA256 checksums are automatically generated for all binaries and included in `checksums.txt`.

Users can verify downloads:
```bash
wget https://github.com/YOUR_USERNAME/dbdiff/releases/download/v1.0.0/checksums.txt
sha256sum -c checksums.txt
```

## Troubleshooting

### Release workflow failed

1. Check GitHub Actions logs: `Actions` tab → `Release` workflow
2. Common issues:
   - Go version mismatch (update `.github/workflows/release.yml`)
   - Missing dependencies (run `go mod tidy`)
   - Build errors (test locally with `make build-all`)

### Tag already exists

```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push origin :refs/tags/v1.0.0

# Create new tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### Need to update a release

1. Delete the release on GitHub (keep the tag)
2. Delete the tag locally and remotely (see above)
3. Make your changes and commit
4. Create the tag again and push

## Release Checklist

Before creating a release:

- [ ] All tests pass (`make test`)
- [ ] Code builds successfully (`make build-all`)
- [ ] README is up to date
- [ ] CHANGELOG is updated (if you have one)
- [ ] Version number follows semantic versioning
- [ ] All changes are committed and pushed
- [ ] Tag message is descriptive

After creating a release:

- [ ] Verify all binaries are attached to GitHub release
- [ ] Test download and installation on at least one platform
- [ ] Verify checksums match
- [ ] Update documentation if needed
- [ ] Announce release (if applicable)

## Example Release Workflow

```bash
# 1. Make changes
vim main.go
git add main.go
git commit -m "Add support for Oracle database"

# 2. Update version references if needed
vim README.md
git add README.md
git commit -m "Update README for v1.2.0"

# 3. Push changes
git push origin main

# 4. Create and push tag
git tag -a v1.2.0 -m "Release v1.2.0 - Oracle support"
git push origin v1.2.0

# 5. Wait for GitHub Actions to complete (~2-3 minutes)

# 6. Verify release at:
# https://github.com/YOUR_USERNAME/dbdiff/releases/tag/v1.2.0
```

## Notes

- The GitHub Actions workflow requires `GITHUB_TOKEN` which is automatically provided
- No additional secrets or configuration needed
- First-time setup: Just push a tag and the workflow runs automatically
- Release notes are auto-generated but can be edited manually on GitHub

