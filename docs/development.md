# Development Guide

This guide covers everything you need to know to develop and contribute to Rackd.

## Prerequisites

### Required Tools

- **Go 1.25+**: The project requires Go 1.25 or later
- **Bun**: JavaScript runtime and package manager for frontend development
- **Make**: Build automation tool
- **Git**: Version control

### Optional Tools

- **golangci-lint**: For code linting
- **gosec**: For security scanning
- **gofumpt**: For enhanced code formatting
- **Docker**: For containerized development and deployment

### Installation

#### Go
```bash
# Install Go 1.25+ from https://golang.org/dl/
# Or using a version manager like g or gvm
```

#### Bun
```bash
# Install Bun
curl -fsSL https://bun.sh/install | bash
```

#### Development Tools
```bash
# Install linting and formatting tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install mvdan.cc/gofumpt@latest
```

## Building from Source

### Quick Start

```bash
# Clone the repository
git clone https://github.com/martinsuchenak/rackd.git
cd rackd

# Install dependencies
go mod download
cd webui && bun install && cd ..

# Build the complete application
make build

# Run the server
./build/rackd server
```

### Build Targets

The project uses a Makefile with several build targets:

#### Core Targets
- `make build` - Build complete application (UI + binary)
- `make binary` - Build Go binary only
- `make ui-build` - Build web UI assets only
- `make clean` - Remove build artifacts

#### Development Targets
- `make dev` - Run in development mode with hot reload
- `make run-server` - Build and run server
- `make fmt` - Format code
- `make lint` - Run linters
- `make validate` - Run all validations (build, test, vet, lint)

#### Testing Targets
- `make test` - Run all tests
- `make test-short` - Run short tests only
- `make test-race` - Run tests with race detector
- `make test-coverage` - Show test coverage

#### Cross-Platform Builds
- `make build-linux` - Build for Linux (amd64, arm64)
- `make build-darwin` - Build for macOS (amd64, arm64)
- `make build-windows` - Build for Windows (amd64)

#### Docker Targets
- `make docker` - Build Docker image
- `make docker-run` - Run Docker container

### Build Process

The build process consists of two main phases:

1. **Frontend Build** (`make ui-build`):
   - Installs Node.js dependencies with Bun
   - Compiles TypeScript to JavaScript
   - Processes CSS with TailwindCSS
   - Builds HTML templates
   - Copies assets to `internal/ui/assets/`

2. **Backend Build** (`make binary`):
   - Compiles Go code with embedded UI assets
   - Includes version information via ldflags
   - Produces single binary with no external dependencies

## Project Structure

```
rackd/
├── api/                    # OpenAPI specifications
├── build/                  # Build output directory
├── cmd/                    # CLI commands and subcommands
│   ├── client/            # Client utilities
│   ├── datacenter/        # Datacenter management commands
│   ├── device/            # Device management commands
│   ├── discovery/         # Discovery commands
│   ├── network/           # Network management commands
│   └── server/            # Server command
├── data/                   # Default data directory (SQLite database)
├── deploy/                 # Deployment configurations
├── docs/                   # Documentation
├── internal/               # Private application code
│   ├── api/               # HTTP API handlers
│   ├── config/            # Configuration management
│   ├── credentials/       # Credential storage and encryption
│   ├── discovery/         # Network discovery logic
│   ├── log/               # Logging utilities
│   ├── mcp/               # Model Context Protocol server
│   ├── model/             # Data models and DTOs
│   ├── server/            # HTTP server setup
│   ├── storage/           # Database layer (SQLite)
│   ├── types/             # Common types
│   ├── ui/                # Embedded UI assets
│   └── worker/            # Background job processing
├── pkg/                    # Public API packages
├── webui/                  # Frontend source code
│   ├── assets/            # Static assets
│   ├── dist/              # Built frontend (generated)
│   ├── scripts/           # Build scripts
│   └── src/               # TypeScript/HTML source
└── main.go                # Application entry point
```

### Key Directories

- **`cmd/`**: Contains CLI command implementations using the `paularlott/cli` framework
- **`internal/`**: Private application code that cannot be imported by external packages
- **`pkg/`**: Public packages that can be imported by other projects
- **`webui/`**: Frontend application built with Alpine.js and TailwindCSS
- **`api/`**: OpenAPI specifications for the REST API

## Testing

### Test Organization

Tests are organized alongside the code they test, following Go conventions:

- Unit tests: `*_test.go` files in the same package
- Integration tests: `integration_test.go` files
- Test utilities: `testutil/` directories where needed

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run only short tests (excludes integration tests)
make test-short

# Run tests with race detection
make test-race

# Run specific package tests
go test ./internal/storage/...

# Run with verbose output
go test -v ./...
```

### Test Categories

1. **Unit Tests**: Test individual functions and methods
2. **Integration Tests**: Test component interactions
3. **API Tests**: Test HTTP endpoints
4. **Storage Tests**: Test database operations

### Writing Tests

Follow these patterns when writing tests:

```go
func TestFunctionName(t *testing.T) {
    // Setup
    
    // Execute
    
    // Assert
    if got != want {
        t.Errorf("FunctionName() = %v, want %v", got, want)
    }
}

func TestFunctionName_ErrorCase(t *testing.T) {
    // Test error conditions
}
```

For integration tests, use the `testing.Short()` check:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Integration test code
}
```

## Contributing

### Development Workflow

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/rackd.git
   cd rackd
   ```

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Write code following the style guide
   - Add tests for new functionality
   - Update documentation as needed

4. **Validate Changes**
   ```bash
   make validate
   ```

5. **Commit and Push**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   git push origin feature/your-feature-name
   ```

6. **Create Pull Request**
   - Open a PR against the main branch
   - Include description of changes
   - Reference any related issues

### Commit Message Format

Use conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes
- `refactor:` - Code refactoring
- `test:` - Test additions or changes
- `chore:` - Build process or auxiliary tool changes

### Code Review Process

1. All changes require review before merging
2. Automated checks must pass (tests, linting, etc.)
3. At least one maintainer approval required
4. Address review feedback promptly

## Code Style

### Go Code Style

Follow standard Go conventions:

- Use `gofmt` and `gofumpt` for formatting
- Follow effective Go guidelines
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

#### Formatting

```bash
# Format code
make fmt

# This runs:
go fmt ./...
gofumpt -w .
```

#### Linting

```bash
# Run linter
make lint

# This runs:
golangci-lint run ./...
```

### Frontend Code Style

- Use TypeScript for type safety
- Follow Alpine.js conventions
- Use TailwindCSS for styling
- Keep components small and focused
- Use meaningful CSS class names
- **Maintain WCAG 2.2 Level AAA compliance** (see Accessibility Guidelines below)

### Accessibility Guidelines

**All UI changes must comply with WCAG 2.2 Level AAA standards.**

#### Required Standards

1. **Color Contrast (1.4.6 AAA)**
   - Normal text: 7:1 contrast ratio minimum
   - Large text (18pt+): 4.5:1 contrast ratio minimum
   - Use contrast checker before committing color changes

2. **Target Size (2.5.5 AAA)**
   - All interactive elements: 44×44 CSS pixels minimum
   - Includes buttons, links, form controls, and custom controls
   - Ensure adequate spacing between adjacent targets (8px minimum)

3. **Focus Indicators (2.4.7 AAA)**
   - All interactive elements must have visible focus indicators
   - Use `focus:outline-none focus:ring-[3px] focus:ring-blue-500` pattern
   - Icon-only buttons need extra prominent focus (4px outline)
   - Test with keyboard-only navigation

4. **Keyboard Navigation (2.1.3 AAA)**
   - All functionality must be keyboard accessible
   - Implement focus traps in modals using `createFocusTrap()` utility
   - Support arrow key navigation where appropriate (e.g., search results)
   - No keyboard traps - users must be able to navigate away

5. **Form Validation (3.3.1 A, 3.3.3 AA)**
   - Link errors to specific fields with `aria-invalid` and `aria-describedby`
   - Display field-specific error messages
   - Use red borders to indicate invalid fields visually
   - Validate before submission and block if invalid

6. **ARIA Attributes (4.1.2 A)**
   - Use semantic HTML first (button, nav, main, etc.)
   - Add ARIA labels to icon-only buttons
   - Use `role="alert"` for error messages
   - Use `aria-live="polite"` for status updates
   - Add `role="status"` to loading indicators

7. **Help Text (3.3.5 AAA)**
   - Provide context-sensitive help for complex fields
   - Use "?" help buttons with `aria-expanded` state
   - Link help text to inputs with `aria-describedby`
   - Keep help text concise and actionable

8. **Reduced Motion (2.3.3 AAA)**
   - Respect `prefers-reduced-motion` media query
   - All animations/transitions must be disableable
   - Already implemented in `styles.css`

9. **Page Titles (2.4.2 A)**
   - Update document title on route changes
   - Format: "Page Name - Rackd"
   - Already implemented in router

10. **Table Accessibility (1.3.1 A)**
    - Add `<caption>` to all data tables (can be `sr-only`)
    - Use `<th scope="col">` for column headers
    - Use `<th scope="row">` for row headers where appropriate

#### Implementation Checklist

Before submitting UI changes, verify:

- [ ] All interactive elements are 44×44px minimum
- [ ] Color contrast meets 7:1 ratio (use contrast checker)
- [ ] Focus indicators visible on all interactive elements
- [ ] Keyboard navigation works (test without mouse)
- [ ] Form validation shows field-specific errors
- [ ] ARIA attributes present where needed
- [ ] Help text provided for complex fields
- [ ] Modals trap focus properly
- [ ] Tables have captions and proper headers
- [ ] Page title updates on navigation

#### Testing Tools

**Automated Testing:**
```bash
# Run accessibility tests (if configured)
cd webui
bun test:a11y
```

**Manual Testing:**
- **axe DevTools**: Browser extension for automated scanning
- **WAVE**: Web accessibility evaluation tool
- **Lighthouse**: Chrome DevTools accessibility audit
- **Keyboard only**: Unplug mouse and navigate with Tab/Enter/Arrows
- **Screen reader**: Test with NVDA (Windows), JAWS (Windows), or VoiceOver (macOS)

**Contrast Checkers:**
- WebAIM Contrast Checker: https://webaim.org/resources/contrastchecker/
- Coolors Contrast Checker: https://coolors.co/contrast-checker

#### Common Patterns

**Button with Icon:**
```html
<button class="p-2 min-w-[44px] min-h-[44px] focus:outline-none focus:ring-[3px] focus:ring-blue-500"
        aria-label="Close dialog">
  <svg class="w-5 h-5" aria-hidden="true">...</svg>
</button>
```

**Form Field with Validation:**
```html
<label for="field-name">Name <span class="sr-only">(required)</span></label>
<input id="field-name"
       :aria-invalid="!!errors.name"
       :aria-describedby="errors.name ? 'field-name-error' : undefined"
       :class="errors.name ? 'border-red-600' : 'border-gray-300'">
<p x-show="errors.name" id="field-name-error" role="alert" x-text="errors.name"></p>
```

**Help Text:**
```html
<label for="field">
  Field Name
  <button type="button" @click="showHelp = !showHelp"
          :aria-expanded="showHelp"
          aria-label="Help for field name">?</button>
</label>
<input id="field" aria-describedby="field-help">
<p x-show="showHelp" id="field-help">Helpful explanation here.</p>
```

**Modal with Focus Trap:**
```typescript
// In component init
this.$watch('showModal', (show: boolean) => {
  if (show) {
    setTimeout(() => {
      const modal = document.querySelector('[role="dialog"]') as HTMLElement;
      if (modal) this.focusTrapCleanup = createFocusTrap(modal);
    }, 50);
  } else {
    this.focusTrapCleanup?.();
    this.focusTrapCleanup = null;
  }
});
```

#### Resources

- **WCAG 2.2 Guidelines**: https://www.w3.org/WAI/WCAG22/quickref/
- **ARIA Authoring Practices**: https://www.w3.org/WAI/ARIA/apg/
- **Accessibility Audit**: See `ACCESSIBILITY_AUDIT.md`
- **Implementation Guides**: See `docs/accessibility-*.md`

### Documentation Style

- Use clear, concise language
- Include code examples where helpful
- Keep documentation up to date with code changes
- Use proper Markdown formatting

### Error Handling

- Always handle errors explicitly
- Use meaningful error messages
- Wrap errors with context when appropriate
- Log errors at appropriate levels

```go
// Good
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process data: %w", err)
}

// Bad
result, _ := someFunction()
```

### Security Considerations

- Never commit secrets or credentials
- Use proper input validation
- Sanitize user inputs
- Follow secure coding practices
- Run security scans regularly

```bash
# Run security scanner
make security
```

## Development Environment

### Environment Variables

Create a `.env` file for local development:

```bash
# Copy example environment file
cp .env.example .env

# Edit with your settings
DATA_DIR=./data
LISTEN_ADDR=:8080
LOG_LEVEL=debug
LOG_FORMAT=text
```

### Database

Rackd uses SQLite for data storage:

- Database file: `./data/rackd.db` (default)
- Migrations run automatically on startup
- No external database server required

### Hot Reload Development

```bash
# Start development server with hot reload
make dev

# This will:
# 1. Build the UI
# 2. Start the server with debug logging
# 3. Watch for changes (manual restart required)
```

### IDE Configuration

#### VS Code

The project includes VS Code settings in `.vscode/settings.json`:

```json
{
    "go.lintTool": "golangci-lint",
    "go.formatTool": "gofumpt"
}
```

Recommended extensions:
- Go extension
- TypeScript extension
- TailwindCSS IntelliSense

### Debugging

#### Go Debugging

Use the built-in Go debugger or IDE debugging features:

```bash
# Run with debug logging
LOG_LEVEL=debug ./build/rackd server

# Use delve for debugging
dlv debug . -- server
```

#### Frontend Debugging

- Use browser developer tools
- Enable debug logging in the application
- Check network requests in browser dev tools

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Ensure Go 1.25+ is installed
   - Run `go mod download` to fetch dependencies
   - Check that Bun is installed for frontend builds

2. **Test Failures**
   - Run tests individually to isolate issues
   - Check for race conditions with `make test-race`
   - Ensure test database is clean

3. **Frontend Issues**
   - Clear `webui/dist/` and `internal/ui/assets/`
   - Reinstall frontend dependencies: `cd webui && bun install`
   - Check browser console for JavaScript errors

4. **Database Issues**
   - Delete database file to reset: `rm -f data/rackd.db*`
   - Check file permissions on data directory
   - Ensure SQLite is available

### Getting Help

- Check existing GitHub issues
- Review documentation in `docs/` directory
- Ask questions in GitHub Discussions
- Join community channels (if available)

## Release Process

### Version Management

- Versions follow semantic versioning (semver)
- Version information embedded during build via ldflags
- Git tags used for releases

### Building Releases

```bash
# Build for all platforms
make build-linux
make build-darwin
make build-windows

# Or use GoReleaser (if configured)
goreleaser release --snapshot --rm-dist
```

This development guide provides everything needed to start contributing to Rackd. For specific feature documentation, see the other guides in the `docs/` directory.