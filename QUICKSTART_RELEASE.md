# Quick Start: Creating Your First Release

This guide will walk you through creating your first GitHub release with pre-built binaries.

## Prerequisites

1. **GitHub Repository**: Push your code to GitHub
   ```bash
   git remote add origin https://github.com/YOUR_USERNAME/dbdiff.git
   git push -u origin main
   ```

2. **Verify GitHub Actions is enabled**: 
   - Go to your repository on GitHub
   - Click "Actions" tab
   - If prompted, enable GitHub Actions

## Step-by-Step: Create Your First Release

### 1. Ensure Everything is Ready

```bash
# Make sure all changes are committed
git status

# If there are uncommitted changes:
git add .
git commit -m "Prepare for v1.0.0 release"
git push origin main
```

### 2. Create a Version Tag

```bash
# Create an annotated tag for version 1.0.0
git tag -a v1.0.0 -m "Initial release v1.0.0"

# Verify the tag was created
git tag -l
```

### 3. Push the Tag to GitHub

```bash
# Push the tag to trigger the release workflow
git push origin v1.0.0
```

### 4. Watch the Build Process

1. Go to your GitHub repository
2. Click the "Actions" tab
3. You should see a "Release" workflow running
4. Click on it to watch the build progress (~2-3 minutes)

### 5. Verify the Release

Once the workflow completes:

1. Go to the "Releases" section of your repository
2. You should see "v1.0.0" release
3. Verify all 6 binaries are attached:
   - `dbdiff-linux-amd64`
   - `dbdiff-linux-arm64`
   - `dbdiff-darwin-amd64`
   - `dbdiff-darwin-arm64`
   - `dbdiff-windows-amd64.exe`
   - `dbdiff-windows-arm64.exe`
   - `checksums.txt`

### 6. Test the Release

Download and test one of the binaries:

```bash
# Example: Download Linux AMD64 binary
wget https://github.com/YOUR_USERNAME/dbdiff/releases/download/v1.0.0/dbdiff-linux-amd64

# Make it executable
chmod +x dbdiff-linux-amd64

# Test it
./dbdiff-linux-amd64 --help
```

## What Happens Automatically?

When you push a tag starting with `v`, GitHub Actions will:

1. ✅ Checkout your code
2. ✅ Set up Go environment
3. ✅ Build binaries for 6 platforms:
   - Linux (AMD64, ARM64)
   - macOS (Intel, Apple Silicon)
   - Windows (AMD64, ARM64)
4. ✅ Generate SHA256 checksums
5. ✅ Create a GitHub Release
6. ✅ Upload all binaries and checksums
7. ✅ Add installation instructions to release notes

## Troubleshooting

### "Tag already exists" error

```bash
# Delete the local tag
git tag -d v1.0.0

# Delete the remote tag (if it was pushed)
git push origin :refs/tags/v1.0.0

# Create the tag again
git tag -a v1.0.0 -m "Initial release v1.0.0"
git push origin v1.0.0
```

### Workflow failed

1. Check the Actions tab for error logs
2. Common fixes:
   - Ensure `go.mod` and `go.sum` are committed
   - Verify `main.go` compiles locally: `go build main.go`
   - Check the workflow file: `.github/workflows/release.yml`

### Need to update a release

1. Go to Releases on GitHub
2. Click "Edit" on the release
3. You can update the description, add/remove files, etc.

Or delete and recreate:
```bash
# Delete the tag
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Delete the release on GitHub (via web UI)

# Make your changes, commit, and create tag again
git tag -a v1.0.0 -m "Initial release v1.0.0"
git push origin v1.0.0
```

## Creating Subsequent Releases

For your next release:

```bash
# Make your changes
git add .
git commit -m "Add new feature"
git push origin main

# Create new version tag
git tag -a v1.1.0 -m "Release v1.1.0 - New feature"
git push origin v1.1.0

# GitHub Actions will automatically build and release!
```

## Version Numbering Guide

Follow [Semantic Versioning](https://semver.org/):

- **v1.0.0** → First stable release
- **v1.0.1** → Bug fix (patch)
- **v1.1.0** → New feature, backwards compatible (minor)
- **v2.0.0** → Breaking changes (major)

Examples:
```bash
git tag -a v1.0.1 -m "Fix connection timeout bug"
git tag -a v1.1.0 -m "Add SQLite support"
git tag -a v2.0.0 -m "New CLI interface (breaking change)"
```

## Pre-releases (Beta/RC)

For testing before official release:

```bash
git tag -a v1.0.0-beta.1 -m "Beta release for testing"
git push origin v1.0.0-beta.1
```

Then mark it as "pre-release" in the GitHub UI.

## Local Testing Before Release

Test building all platforms locally:

```bash
# Build all binaries
make build-all

# Check the build/ directory
ls -lh build/

# Test the Linux binary
./build/dbdiff-linux-amd64 --help

# Clean up
make clean
```

## Summary

**To create a release, you only need 2 commands:**

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

Everything else is automated! 🚀

## Next Steps

- Share your release with users
- Update documentation with download links
- Consider creating a changelog (CHANGELOG.md)
- Set up release notifications
- Add badges to README showing latest version

## Need Help?

- Check `.github/workflows/release.yml` for workflow configuration
- See `RELEASE.md` for detailed release documentation
- Review GitHub Actions logs for build errors
- Test locally with `make build-all` before creating tags

