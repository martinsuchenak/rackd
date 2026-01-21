# Release Management

This document defines versioning strategy, release process, and automation for Rackd.

## 1. Versioning Strategy

### 1.1 Semantic Versioning Rules

**Format:** `MAJOR.MINOR.PATCH`

**MAJOR Version:**
- Incompatible API changes
- Breaking database schema changes
- Removal of deprecated features
- Major architectural changes

**MINOR Version:**
- New backward-compatible features
- Non-breaking schema changes (additions)
- New configuration options
- Performance improvements

**PATCH Version:**
- Bug fixes
- Security fixes
- Non-breaking changes
- Documentation updates

**Examples:**
- `v1.0.0` → `v1.0.1` (Patch: Bug fix)
- `v1.0.1` → `v1.1.0` (Minor: New feature)
- `v1.1.0` → `v2.0.0` (Major: Breaking changes)

### 1.2 Pre-Release Versions

**Format:** `MAJOR.MINOR.PATCH-PRERELEASE`

**Types:**

| Type | Description | Usage |
|------|-------------|-------|
| `alpha` | Early development, unstable | Feature development |
| `beta` | Feature complete, testing needed | Public testing |
| `rc` (release candidate) | Feature complete, ready for testing | Pre-release testing |

**Examples:**
- `v1.2.0-alpha.1` (First alpha of v1.2.0)
- `v1.2.0-beta.1` (First beta of v1.2.0)
- `v1.2.0-rc.1` (First release candidate)

**Pre-Release Ordering:**

```bash
v1.2.0-alpha.1
v1.2.0-alpha.2
v1.2.0-beta.1
v1.2.0-rc.1
v1.2.0-rc.2
v1.2.0 (final)
```

### 1.3 Build Metadata

**Format:** `VERSION+BUILDMETA`

**Components:**

- **Version**: Semantic version (e.g., `v1.2.0`)
- **Build timestamp**: UTC Unix timestamp (e.g., `20240120100000`)
- **Commit hash**: First 7 characters of git commit (e.g., `a1b2c3d`)

**Examples:**
- `v1.2.0+20240120100000.a1b2c3d`
- `v1.2.0-rc.1+20240120080000.4e5f6a7`

**Git Tag Format:**

```bash
# Standard release tag
v1.2.0

# Pre-release tag
v1.2.0-rc.1

# Build metadata tag
v1.2.0+build.1234567890
```

### 1.4 Version Comparison

**Comparison Rules:**

1. **Pre-release < release**: `v1.2.0-rc.1` < `v1.2.0`
2. **Older < newer**: `v1.1.0` < `v1.2.0`
3. **Build metadata ignored**: `v1.2.0+build.X` == `v1.2.0`

**Implementation:**

```go
package version

import (
    "regexp"
    "strconv"
    "strings"
)

type Version struct {
    Major       int
    Minor       int
    Patch       int
    PreRelease string
    BuildMeta   string
}

func Parse(v string) (*Version, error) {
    // Remove build metadata
    v = strings.Split(v, "+")[0]

    // Parse version and pre-release
    parts := strings.SplitN(v, "-", 2)

    versionStr := parts[0]
    var preRelease string
    if len(parts) > 1 {
        preRelease = parts[1]
    }

    // Parse semantic version
    numbers := strings.Split(versionStr, ".")
    if len(numbers) != 3 {
        return nil, &VersionError{Message: "Invalid version format"}
    }

    major, err := strconv.Atoi(numbers[0])
    if err != nil {
        return nil, err
    }

    minor, err := strconv.Atoi(numbers[1])
    if err != nil {
        return nil, err
    }

    patch, err := strconv.Atoi(numbers[2])
    if err != nil {
        return nil, err
    }

    return &Version{
        Major:       major,
        Minor:       minor,
        Patch:       patch,
        PreRelease: preRelease,
    }, nil
}

func Compare(v1, v2 string) int {
    ver1, err := Parse(v1)
    if err != nil {
        return 0
    }

    ver2, err := Parse(v2)
    if err != nil {
        return 0
    }

    // Compare major version
    if ver1.Major != ver2.Major {
        return ver1.Major - ver2.Major
    }

    // Compare minor version
    if ver1.Minor != ver2.Minor {
        return ver1.Minor - ver2.Minor
    }

    // Compare patch version
    if ver1.Patch != ver2.Patch {
        return ver1.Patch - ver2.Patch
    }

    // Handle pre-release
    pre1 := preReleaseOrder(ver1.PreRelease)
    pre2 := preReleaseOrder(ver2.PreRelease)

    if pre1 != pre2 {
        return pre1 - pre2
    }

    return 0
}

func preReleaseOrder(preRelease string) int {
    if preRelease == "" {
        return 0 // Release
    }
    if strings.HasPrefix(preRelease, "rc") {
        return -1 // RC < release
    }
    if strings.HasPrefix(preRelease, "beta") {
        return -2 // Beta < RC
    }
    if strings.HasPrefix(preRelease, "alpha") {
        return -3 // Alpha < Beta
    }
    return -100 // Unknown
}

func (v *Version) String() string {
    version := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
    if v.PreRelease != "" {
        version += "-" + v.PreRelease
    }
    if v.BuildMeta != "" {
        version += "+" + v.BuildMeta
    }
    return version
}
```

---

## 2. Changelog Format

### 2.1 Keep a Changelog Format

**Template:**

```markdown
# [Unreleased]

### Added
- New feature 1
- New feature 2
- ...

### Changed
- Changed feature 1
- Changed feature 2
- ...

### Deprecated
- Deprecated feature 1
- Deprecated feature 2
- ...

### Removed
- Removed feature 1
- Removed feature 2
- ...

### Fixed
- Bug fix 1
- Bug fix 2
- ...

### Security
- Security fix 1
- Security fix 2
- ...
```

**Section Descriptions:**

| Section | Description | When to Use |
|---------|-------------|-------------|
| **Added** | New features added | New functionality |
| **Changed** | Changes to existing features | Behavior or implementation changes |
| **Deprecated** | Features to be removed | Announce future removal |
| **Removed** | Features removed in this version | Actual removal |
| **Fixed** | Bug fixes | Bug fixes and corrections |
| **Security** | Security-related fixes | Security patches |

### 2.2 Pull Request Template

**Template:**

```markdown
## Description
[Brief description of changes]

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Related Issues
Fixes #123, #456
Relates #789

## How Has This Been Tested?

### Unit Tests
- [ ] Unit tests added/updated
- [ ] All unit tests passing

### Integration Tests
- [ ] Integration tests added/updated
- [ ] All integration tests passing

### Manual Testing
- [ ] Tested on macOS
- [ ] Tested on Linux
- [ ] Tested on Windows (if applicable)

## Checklist:
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published in downstream modules

## Changelog Entry

### Added
[Description of added features]

### Changed
[Description of changed features]

### Deprecated
[Description of deprecated features]

### Removed
[Description of removed features]

### Fixed
[Description of fixed bugs]
```

### 2.3 Automated Changelog Generation

**Script:**

```bash
#!/bin/bash
# scripts/generate-changelog.sh

# Get last tag
LAST_TAG=$(git describe --tags --abbrev=0)

# Get commits since last tag
COMMITS=$(git log ${LAST_TAG}..HEAD --pretty=format:"%h|%s|%b|%an")

# Parse commits
echo "# [$(git describe --tags --abbrev=0)] ($(date +%Y-%m-%d))"
echo ""

# Categorize commits
echo "### Added"
echo "$COMMITS" | grep "feat:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo "### Fixed"
echo "$COMMITS" | grep "fix:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo "### Changed"
echo "$COMMITS" | grep "chore:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq
```

**Commit Message Format:**

```
type(scope): description

Types:
- feat: New feature
- fix: Bug fix
- docs: Documentation change
- style: Code style change (formatting)
- refactor: Code refactoring
- test: Adding or updating tests
- chore: Maintenance task

Examples:
feat(api): add rate limiting middleware
fix(storage): resolve database lock issue
docs(readme): update installation instructions
```

---

## 3. Release Process

### 3.1 Branching Strategy

**Branch Structure:**

```
main (production)
  ├─> v1.0.0
  ├─> v1.1.0
  └─> v1.2.0

develop (staging)
  ├─> merges to main for releases

release/v1.2.0 (release preparation)
  └─> merged to main for release

feature/xyz (feature branch)
  └─> merged to develop
```

**Branch Naming Conventions:**

- `main` - Production branch
- `develop` - Staging/development branch
- `release/vX.Y.Z` - Release preparation branch
- `feature/feature-name` - Feature development
- `hotfix/vX.Y.Z` - Hotfix for release

### 3.2 PR Review Requirements

**Checklist:**

- [ ] All automated tests passing
- [ ] Code reviewed by at least 1 maintainer
- [ ] Documentation updated
- [ ] Changelog entry added
- [ ] Breaking changes documented
- [ ] Security considerations reviewed
- [ ] Performance impact assessed
- [ ] Backward compatibility verified
- [ ] Migration path documented

**Review Categories:**

1. **Code Quality**
   - Follows project style guidelines
   - No compiler warnings
   - Proper error handling
   - Adequate test coverage

2. **Functionality**
   - Implements intended feature
   - Handles edge cases
   - No regressions

3. **Documentation**
   - API documentation updated
   - User documentation updated
   - Code comments adequate
   - Examples provided

4. **Security**
   - No security vulnerabilities
   - Proper input validation
   - No secrets in code
   - Dependencies reviewed

### 3.3 Release Checklist

**Pre-Release Checklist:**

```markdown
## Release vX.Y.Z Checklist

### Code
- [ ] All features implemented
- [ ] All tests passing
- [ ] Code reviewed
- [ ] Linting clean
- [ ] No TODO comments for release blockers

### Documentation
- [ ] README updated
- [ ] API documentation updated
- [ ] User guide updated
- [ ] Migration guide created (if needed)
- [ ] Breaking changes documented
- [ ] Upgrade guide updated

### Testing
- [ ] Unit tests passing (> 80% coverage)
- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] Manual testing completed
- [ ] Performance testing completed
- [ ] Security testing completed

### Build
- [ ] All platforms built successfully
- [ ] Docker image built
- [ ] Release artifacts generated
- [ ] Checksums verified
- [ ] Packages created (deb, rpm)

### Preparation
- [ ] Release tag created
- [ ] Release branch merged
- [ ] Changelog generated
- [ ] Release notes written
- [ ] Announcement prepared

### Verification
- [ ] Artifacts uploaded
- [ ] Docker images pushed
- [ ] Homebrew formula updated
- [ ] Documentation published
- [ ] Announcement sent
```

### 3.4 Release Candidate Process

**RC Cycle:**

```
v1.2.0-alpha.1 → v1.2.0-alpha.2 → v1.2.0-beta.1
→ v1.2.0-rc.1 → v1.2.0-rc.2 → v1.2.0 (final)
```

**RC Testing:**

1. **Create RC**
   ```bash
   # Create RC tag
   git tag -a v1.2.0-rc.1 -m "Release candidate 1 for v1.2.0"

   # Push tag
   git push origin v1.2.0-rc.1
   ```

2. **Test RC**
   - Deploy to staging environment
   - Run full test suite
   - Manual testing
   - Performance testing
   - Upgrade testing

3. **Address Issues**
   - Create hotfix branch
   - Fix issues
   - Merge to release branch
   - Tag new RC

4. **Final Release**
   - After successful RC testing, create final tag

### 3.5 Final Release Process

**Step-by-Step:**

```bash
# 1. Create release branch from develop
git checkout develop
git pull
git checkout -b release/v1.2.0

# 2. Update version
sed -i 's/version = "dev"/version = "v1.2.0"/' version.go
git commit -am "Bump version to v1.2.0"

# 3. Merge to develop
git checkout develop
git merge --no-ff release/v1.2.0
git push origin develop

# 4. Merge to main
git checkout main
git merge --no-ff develop
git push origin main

# 5. Tag release
git tag -a v1.2.0 -m "Release v1.2.0"

# 6. Push tag
git push origin v1.2.0

# 7. Create release notes
./scripts/generate-release-notes.sh v1.2.0 > release-notes-v1.2.0.md

# 8. Build release artifacts
make release VERSION=v1.2.0
```

---

## 4. Release Artifacts

### 4.1 Binary Builds

**Platform Matrix:**

| Platform | Architecture | Binary Name | CGO | Target |
|----------|-------------|-------------|------|--------|
| Linux | amd64 | rackd_1.2.0_linux_amd64 | 0 | linux/amd64 |
| Linux | arm64 | rackd_1.2.0_linux_arm64 | 0 | linux/arm64 |
| macOS | amd64 | rackd_1.2.0_darwin_amd64 | 0 | darwin/amd64 |
| macOS | arm64 | rackd_1.2.0_darwin_arm64 | 0 | darwin/arm64 |
| Windows | amd64 | rackd_1.2.0_windows_amd64.exe | 0 | windows/amd64 |

**Build Commands:**

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.version=v1.2.0 -X main.commit=abcdef1234567" \
  -o dist/rackd_1.2.0_linux_amd64 \
  .

# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -ldflags="-s -w -X main.version=v1.2.0" \
  -o dist/rackd_1.2.0_linux_arm64 \
  .

# macOS amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
  -ldflags="-s -w -X main.version=v1.2.0" \
  -o dist/rackd_1.2.0_darwin_amd64 \
  .

# macOS arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
  -ldflags="-s -w -X main.version=v1.2.0" \
  -o dist/rackd_1.2.0_darwin_arm64 \
  .

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -ldflags="-s -w -H windowsgui -X main.version=v1.2.0" \
  -o dist/rackd_1.2.0_windows_amd64.exe \
  .
```

### 4.2 Docker Images

**Multi-Architecture Build:**

```dockerfile
# Dockerfile (build stage)
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o /app/rackd \
  .

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/rackd /usr/local/bin/rackd

RUN mkdir -p /data

ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/datacenters || exit 1

ENTRYPOINT ["rackd"]
CMD ["server"]
```

**Build Command:**

```bash
# Build for multiple architectures
docker buildx build \
  --platform linux/amd64 \
  --platform linux/arm64 \
  --platform darwin/amd64 \
  --platform darwin/arm64 \
  -t martinsuchenak/rackd:v1.2.0 \
  --push .
```

### 4.3 Checksums and Signatures

**Generating Checksums:**

```bash
#!/bin/bash
# scripts/generate-checksums.sh

VERSION=$1
DIST_DIR="dist"

echo "Generating checksums for Rackd $VERSION"

# Generate SHA256 checksums
echo "SHA256 Checksums:" > checksums.txt
for file in ${DIST_DIR}/*; do
    sha256=$(sha256sum $file | awk '{print $1}')
    basename=$(basename $file)
    echo "$sha256  $basename" >> checksums.txt
done

# Generate SHA512 checksums
echo "SHA512 Checksums:" >> checksums.txt
for file in ${DIST_DIR}/*; do
    sha512=$(sha512sum $file | awk '{print $1}')
    basename=$(basename $file)
    echo "$sha512  $basename" >> checksums.txt
done

echo "Checksums generated: checksums.txt"
```

**Sample Checksum File:**

```
SHA256 Checksums:
a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8  rackd_1.2.0_linux_amd64
f1e2d3c4b5a6f7e8d9c0e1f2a3b4c5d6e7f8  rackd_1.2.0_darwin_amd64

SHA512 Checksums:
abc123...  rackd_1.2.0_linux_amd64
def456...  rackd_1.2.0_darwin_amd64
```

**GPG Signing:**

```bash
# Sign checksums file
gpg --default-key martin@martinsuchenak.com --armor --detach-sign --output checksums.txt.sig checksums.txt

# Verify signature
gpg --verify checksums.txt.sig checksums.txt
```

### 4.4 Homebrew Formula

**Formula Template:**

```ruby
# Formula/rackd.rb
class Rackd < Formula
  desc "IP Address Management (IPAM) and Device Inventory System"
  homepage "https://github.com/martinsuchenak/rackd"
  url "https://github.com/martinsuchenak/rackd/archive/refs/tags/v1.2.0.tar.gz"
  sha256 "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8"

  depends_on "sqlite"

  def install
    bin.install "rackd"
    etc.install "config", "rackd"
    etc.install "completion", "rackd"

  test do
    system "#{bin}/rackd version"
  end
end
```

**Update Process:**

```bash
# Clone tap
git clone https://github.com/martinsuchenak/homebrew-tap.git
cd homebrew-tap

# Update formula
cp Formula/rackd.rb Formula/rackd.rb

# Commit and push
git add Formula/rackd.rb
git commit -m "rackd 1.2.0"
git push origin main
```

### 4.5 Debian/RPM Packages

**Debian Package:**

```bash
# Create deb package
fpm -s dir \
  -n rackd \
  -v 1.2.0 \
  -a amd64 \
  -t deb \
  --deb-priority optional \
  --deb-compression bzip2 \
  --url https://github.com/martinsuchenak/rackd \
  --description "IPAM and Device Inventory System" \
  --license MIT \
  --category admin \
  dist/rackd usr/local/bin
```

**RPM Package:**

```bash
# Create rpm package
fpm -s dir \
  -n rackd \
  -v 1.2.0 \
  -a amd64 \
  -t rpm \
  --rpm-os linux \
  --url https://github.com/martinsuchenak/rackd \
  --description "IPAM and Device Inventory System" \
  --license MIT \
  dist/rackd usr/local/bin
```

---

## 5. Deprecation Policy

### 5.1 Deprecation Timeline

**Minimum Notice Period:**

| Change Type | Minimum Notice | Examples |
|-------------|----------------|----------|
| API endpoint removal | 3 months | 1 major version |
| API field removal | 2 months | 1 major version |
| API behavior change | 2 months | 1 major version |
| Configuration option removal | 2 versions | 1 minor + 1 major |
| CLI command removal | 1 version | 1 major version |
| Database schema change | 2 versions | 1 minor + 1 major |

**Deprecation Stages:**

```
Stage 1: Deprecation announced (v1.2.0)
  ↓
Stage 2: Alternative provided (v1.3.0)
  ↓
Stage 3: Warning issued (v1.4.0)
  ↓
Stage 4: Removal scheduled (v2.0.0-rc.1)
  ↓
Stage 5: Feature removed (v2.0.0)
```

### 5.2 Deprecation Notices

**In-Code Deprecation:**

```go
// Deprecated function with warning
func OldFunction() error {
    log.Warn("OldFunction is deprecated and will be removed in v2.0.0. Use NewFunction instead")
    // ... implementation
}

// Deprecated API endpoint
func (h *Handler) oldEndpoint(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("X-Rackd-Deprecated", "true")
    w.Header().Set("X-Rackd-Deprecation-Version", "v2.0.0")
    w.Header().Set("X-Rackd-Deprecation-Date", "2025-06-01")
    w.Header().Set("X-Rackd-Deprecation-Message", "Use /api/v2/devices instead")
    w.Header().Set("X-Rackd-Deprecation-Link", "https://github.com/martinsuchenak/rackd/blob/main/docs/migration.md")

    h.writeJSON(w, http.StatusOK, data)
}
```

**CLI Deprecation Warning:**

```go
func (c *CLI) oldCommand() {
    fmt.Fprintf(os.Stderr,
        "WARNING: 'old-command' is deprecated and will be removed in v2.0.0 (June 2025)\n")
    fmt.Fprintf(os.Stderr,
        "Use 'new-command' instead.\n")
    fmt.Fprintf(os.Stderr,
        "See https://github.com/martinsuchenak/rackd/docs/cli.md for details.\n")

    // ... continue with command
}
```

### 5.3 Removal Checklist

**Before Removing:**

- [ ] Deprecation announced in previous release
- [ ] Alternative feature available
- [ ] Migration guide published
- [ ] Users notified (via multiple channels)
- [ ] Sufficient time passed (minimum 3 months)

**Removal Process:**

```bash
# 1. Add removal notice to changelog
cat >> CHANGELOG.md <<EOF
### Removed
- Old feature removed (deprecated since v1.2.0)
EOF

# 2. Update documentation
# Remove from API docs
# Add migration guide

# 3. Create new release
git tag -a v2.0.0 -m "Release v2.0.0"
```

---

## 6. Release Automation

### 6.1 CI/CD Pipeline Configuration

**GitHub Actions Workflow:**

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: make test

      - name: Run linting
        run: make lint

      - name: Build binaries
        run: make release-build

      - name: Build Docker image
        run: make docker-build

      - name: Generate checksums
        run: ./scripts/generate-checksums.sh v1.2.0

      - name: Generate changelog
        run: ./scripts/generate-changelog.sh v1.2.0

      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/*
            checksums.txt
          generate_release_notes: true
          body_path: release-notes-v1.2.0.md
          prerelease: false
          tag_name: ${{ github.ref }}
          token: ${{ secrets.GITHUB_TOKEN }}
```

### 6.2 Automated Testing Gates

**Gate Configuration:**

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    strategy:
      matrix:
        go-version: ['1.25']

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: ${{ runner.os }}-go-

      - name: Download dependencies
        run: go mod download

      - name: Build UI
        run: make ui-build

      - name: Run tests
        run: make test

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
```

### 6.3 Security Scanning

**Dependency Scanning:**

```yaml
# .github/workflows/security.yml
name: Security Scan

on:
  push:
    branches: [main, develop]
  pull_request:

jobs:
  security:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Run Gosec
        uses: securego/gosec-action@master

      - name: Run GolangCI-Lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

      - name: Run Trivy
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
```

### 6.4 Staging Deployment

**Automated Deployment to Staging:**

```yaml
# .github/workflows/deploy-staging.yml
name: Deploy to Staging

on:
  push:
    branches: [develop]

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Build
        run: make build

      - name: Deploy
        run: |
          curl -X POST \
            -H "Authorization: Bearer ${{ secrets.STAGING_TOKEN }}" \
            -H "Content-Type: application/json" \
            ${{ secrets.STAGING_URL }}/deploy \
            -d '{"version":"develop","sha":"${{ github.sha }}"}'
```

### 6.5 Production Deployment

**Manual Production Deployment:**

```bash
# 1. Verify staging deployment
curl -f https://staging.rackd.com/api/datacenters || exit 1

# 2. Run smoke tests
./scripts/smoke-test.sh https://staging.rackd.com

# 3. Tag release
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# 4. Deploy to production
./scripts/deploy.sh production v1.2.0

# 5. Verify production deployment
curl -f https://rackd.com/api/datacenters || exit 1
./scripts/smoke-test.sh https://rackd.com
```

---

## 7. Post-Release

### 7.1 Monitoring for Regressions

**Post-Release Monitoring:**

```yaml
# .github/workflows/monitoring.yml
name: Post-Release Monitoring

on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours

jobs:
  monitor:
    runs-on: ubuntu-latest

    steps:
      - name: Check error rates
        run: ./scripts/check-error-rates.sh

      - name: Check performance
        run: ./scripts/check-performance.sh

      - name: Alert on issues
        if: failure()
        run: ./scripts/send-alert.sh
```

### 7.2 User Communication

**Release Announcement Channels:**

1. **GitHub Release**
   - Create release with notes
   - Tag commit
   - Attach artifacts

2. **Mailing List**
   - Send announcement to subscribers
   - Include changelog
   - Upgrade instructions

3. **Twitter/X**
   - Post release announcement
   - Link to release notes

4. **Blog Post**
   - Detailed release blog
   - Feature highlights
   - Upgrade guide

**Announcement Template:**

```markdown
# Rackd v1.2.0 Released

We're excited to announce Rackd v1.2.0! This release includes new features, bug fixes, and improvements.

## What's New

### New Features
- Feature 1: Description
- Feature 2: Description

### Improvements
- Improvement 1: Description
- Improvement 2: Description

### Bug Fixes
- Bug fix 1: Description
- Bug fix 2: Description

## Upgrading

### From v1.1.x
Run: `rackd upgrade v1.2.0`

See [upgrade guide](/docs/upgrade-migration.html) for detailed instructions.

### Known Issues
- Issue 1: Description
- Workaround: Solution

## Downloads

- [Linux (amd64)](link)
- [Linux (arm64)](link)
- [macOS (amd64)](link)
- [macOS (arm64)](link)
- [Windows (amd64)](link)
- [Docker](link)
- [Homebrew](link)

## Documentation

- [Release Notes](/releases/v1.2.0.html)
- [Documentation](/docs/)
- [API Reference](/api/reference.html)
- [Upgrade Guide](/docs/upgrade-migration.html)

## Support

If you encounter issues, please:
- Check the [troubleshooting guide](/docs/troubleshooting.html)
- [Report bugs](https://github.com/martinsuchenak/rackd/issues)
- Join the [community discussion](https://github.com/martinsuchenak/rackd/discussions)

## Next Release

The next release (v1.3.0) is planned for Q2 2024 and will include:
- Planned feature 1
- Planned feature 2

Thanks to all contributors!
```

### 7.3 Feedback Collection

**Post-Release Survey:**

```yaml
# User feedback form
questions:
  - How satisfied are you with this release?
  - What features do you use most?
  - What would you like to see improved?
  - Did you encounter any bugs?
  - How likely are you to recommend Rackd?

  channels:
    - GitHub Discussions
    - Email survey
    - In-app feedback
```

**Issue Triage:**

```bash
# Script to categorize issues
# scripts/triage-issues.sh

# Get recent issues
ISSUES=$(gh issue list --limit 100 --state open)

# Categorize
for issue in $ISSUES; do
    # Check labels
    labels=$(echo $issue | jq '.labels[]')

    if [[ $labels == *"bug"* ]]; then
        echo "BUG: $issue"
    elif [[ $labels == *"enhancement"* ]]; then
        echo "ENHANCEMENT: $issue"
    elif [[ $labels == *"help"* ]]; then
        echo "HELP WANTED: $issue"
    else
        echo "NEEDS TRIAGE: $issue"
    fi
done
```

### 7.4 Hotfix Procedures

**Hotfix Branch:**

```bash
# 1. Create hotfix branch from release tag
git checkout -b hotfix/v1.2.1 v1.2.0

# 2. Fix issue
# Make changes

# 3. Commit fix
git commit -am "Fix critical security issue"

# 4. Tag hotfix
git tag -a v1.2.1 -m "Hotfix v1.2.1"

# 5. Push hotfix
git push origin hotfix/v1.2.1
git push origin v1.2.1

# 6. Merge back to main and develop
git checkout main
git merge --no-ff hotfix/v1.2.1
git push origin main

git checkout develop
git merge --no-ff main
git push origin develop
```

---

## 8. Release Tools

### 8.1 Release CLI Commands

```bash
# Prepare release
rackd release prepare v1.2.0

# Build release
rackd release build v1.2.0

# Create tag
rackd release tag v1.2.0

# Generate release notes
rackd release notes v1.2.0

# Create GitHub release
rackd release publish v1.2.0

# All-in-one
rackd release create v1.2.0
```

**Release Create Command:**

```bash
#!/bin/bash
# release.sh

VERSION=$1

echo "Creating release $VERSION"

# 1. Update version
sed -i "s/version = \".*\"/version = \"$VERSION\"/" version.go
git commit -am "Bump version to $VERSION"

# 2. Build artifacts
make release-build VERSION=$VERSION

# 3. Generate checksums
./scripts/generate-checksums.sh $VERSION

# 4. Generate changelog
./scripts/generate-changelog.sh $VERSION

# 5. Tag release
git tag -a $VERSION -m "Release $VERSION"
git push origin $VERSION

# 6. Create GitHub release
gh release create $VERSION \
  --notes-file release-notes-$VERSION.md \
  --title "Rackd $VERSION" \
  --dist dist/ \
  --verify-tag

echo "Release $VERSION created successfully"
```

### 8.2 Version Bump Script

```bash
#!/bin/bash
# scripts/bump-version.sh

TYPE=$1 # major, minor, patch

# Get current version
VERSION=$(git describe --tags --abbrev=0 | sed 's/^v//')

# Parse version
MAJOR=$(echo $VERSION | cut -d. -f1)
MINOR=$(echo $VERSION | cut -d. -f2)
PATCH=$(echo $VERSION | cut -d. -f3)

# Bump version
case $TYPE in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"

echo "Bumping version from $VERSION to $NEW_VERSION"

# Update version file
sed -i "s/version = \".*\"/version = \"${NEW_VERSION}\"/" version.go

# Commit
git commit -am "Bump version to ${NEW_VERSION}"
git tag -a "v${NEW_VERSION}" -m "Version ${NEW_VERSION}"

echo "Version bumped to v${NEW_VERSION}"
```

**Usage:**

```bash
# Bump patch version
./scripts/bump-version.sh patch

# Bump minor version
./scripts/bump-version.sh minor

# Bump major version
./scripts/bump-version.sh major
```

### 8.3 Changelog Generator

```bash
#!/bin/bash
# scripts/generate-changelog.sh

VERSION=$1
PREV_TAG=$(git describe --tags --abbrev=0 HEAD~1)

echo "# Rackd $VERSION ($(date +%Y-%m-%d))"
echo ""

# Get commits since last tag
COMMITS=$(git log ${PREV_TAG}..HEAD --pretty=format:"%s|%b|%an")

# Categorize commits
echo "### Added"
echo "$COMMITS" | grep "feat:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo "### Changed"
echo "$COMMITS" | grep "chore:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo "### Fixed"
echo "$COMMITS" | grep "fix:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo "### Security"
echo "$COMMITS" | grep "security:" | sed 's/^.*|//' | sed 's/^/- /' | sort | uniq

echo ""
echo "## Upgrade Information"
echo "### From $(git describe --tags --abbrev=0 HEAD~1)"
echo "Run \`rackd upgrade $VERSION\` to upgrade from the previous version."
echo ""
echo "See [upgrade guide](docs/upgrade-migration.html) for detailed migration instructions."
```

### 8.4 Release Verification

**Post-Release Verification Checklist:**

```bash
#!/bin/bash
# scripts/verify-release.sh

VERSION=$1

echo "Verifying release $VERSION..."

# 1. Check tag exists
if ! git rev-parse "v${VERSION}" >/dev/null 2>&1; then
    echo "ERROR: Tag v${VERSION} does not exist"
    exit 1
fi
echo "✓ Tag exists"

# 2. Check GitHub release
if ! gh release view "v${VERSION}" >/dev/null 2>&1; then
    echo "ERROR: GitHub release v${VERSION} does not exist"
    exit 1
fi
echo "✓ GitHub release exists"

# 3. Download and verify artifacts
echo "Downloading and verifying artifacts..."
gh release download "v${VERSION}" --dir /tmp/rackd-${VERSION}

# Verify checksums
cd /tmp/rackd-${VERSION}
sha256sum -c checksums.txt
if [ $? -eq 0 ]; then
    echo "✓ Checksums verified"
else
    echo "ERROR: Checksums do not match"
    exit 1
fi

# 4. Run basic tests
echo "Running basic tests..."
./dist/rackd_${VERSION}_linux_amd64 version
if [ $? -eq 0 ]; then
    echo "✓ Binary works"
else
    echo "ERROR: Binary failed to run"
    exit 1
fi

echo ""
echo "All verification checks passed!"
```
