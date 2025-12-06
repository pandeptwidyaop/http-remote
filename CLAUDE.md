# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HTTP Remote is a DevOps tool for remote deployment and command execution on private servers over HTTP/HTTPS. It's designed for servers accessible only through ports 80/443 without VPN. The application is written in Go and provides both a Web UI and REST API for managing applications, commands, and deployments.

**Key Features:**
- Token-based API authentication for CI/CD integration
- Session-based Web UI authentication
- Real-time command output streaming via SSE (Server-Sent Events)
- SQLite database with versioned migrations
- Single binary deployment with embedded assets (HTML, CSS, JS)
- Rate limiting on authentication and deployment endpoints
- Audit logging for security-sensitive operations

## Build & Development Commands

### Building
```bash
make build                  # Build for current platform (requires CGO_ENABLED=1)
make build-linux-amd64     # Cross-compile for Linux AMD64 (requires musl-cross toolchain)
make build-linux-arm64     # Cross-compile for Linux ARM64 (requires musl-cross toolchain)
make build-all             # Build for all platforms
```

**Important:** This project uses SQLite with CGO, so `CGO_ENABLED=1` is required. Cross-compilation needs musl-cross toolchain: `brew install FiloSottile/musl-cross/musl-cross`

### Testing
```bash
make test                        # Run all tests
make test-race                   # Run with race detector
make test-coverage               # Generate HTML coverage report
make test-coverage-summary       # Show coverage percentage only
make test-ci                     # CI mode (race + coverage)
make test-database              # Test database package only
make test-handlers              # Test handlers package only
make test-services              # Test services package only
```

### Running
```bash
make run                    # Build and run with default config
make dev                    # Run with hot reload (auto-installs air if needed)
./http-remote -config path/to/config.yaml    # Run with custom config
./http-remote upgrade       # Check and upgrade to latest version
./http-remote version       # Show version info
```

### Code Quality
```bash
make fmt      # Format code with gofmt
make lint     # Run golangci-lint (must be installed separately)
make tidy     # Tidy go.mod dependencies
```

### Docker
```bash
make docker-build    # Build Docker image
make docker-run      # Run in Docker with data volume
```

## Architecture

### Application Structure

The application follows a layered architecture:

1. **Entry Point** ([cmd/server/main.go](cmd/server/main.go:1-88))
   - Handles subcommands (`upgrade`, `version`)
   - Loads configuration from YAML
   - Initializes database and runs migrations
   - Creates service instances (auth, app, executor, audit)
   - Bootstraps admin user
   - Starts Gin HTTP server

2. **Router** ([internal/router/router.go](internal/router/router.go:1-129))
   - Configures Gin engine with middleware
   - Loads HTML templates from embedded filesystem
   - Serves static files from embedded filesystem
   - Defines all routes with rate limiters:
     - Login endpoints: 5 req/min
     - API endpoints: 60 req/min
     - Deploy endpoints: 30 req/min
   - Separates public (token auth) and protected (session auth) routes

3. **Services Layer**
   - **AuthService** ([internal/services/auth.go](internal/services/auth.go)): Session management, bcrypt password hashing, admin user provisioning
   - **AppService** ([internal/services/apps.go](internal/services/apps.go)): CRUD operations for apps and commands, token generation/regeneration
   - **ExecutorService** ([internal/services/executor.go](internal/services/executor.go:1-335)): Command execution with context timeout, output streaming to subscribers via channels, concurrent stdout/stderr capture
   - **AuditService** ([internal/services/audit.go](internal/services/audit.go)): Logs user actions (login, logout, command execution, CRUD operations)

4. **Handlers Layer** ([internal/handlers/](internal/handlers/))
   - **AuthHandler**: Login/logout for Web UI and API
   - **AppHandler**: App CRUD, token regeneration
   - **CommandHandler**: Command CRUD and execution
   - **DeployHandler**: Public token-based deployment endpoint
   - **StreamHandler**: SSE endpoint for real-time output
   - **WebHandler**: HTML page rendering for dashboard
   - **AuditHandler**: Audit log viewing

5. **Database** ([internal/database/](internal/database/))
   - SQLite with mattn/go-sqlite3 driver
   - Schema defined in [migrations.go](internal/database/migrations.go:1-325)
   - Versioned migration system with tracking table
   - Migrations are idempotent and backward-compatible
   - Foreign key constraints handled carefully (see executions.user_id migration)

6. **Assets** ([internal/assets/](internal/assets/))
   - Web assets (HTML, CSS, JS) embedded via `//go:embed` directive
   - Templates and static files served from embedded filesystem
   - No need to distribute `web/` directory alongside binary

### Key Design Patterns

**Dual Authentication:**
- Web UI uses session cookies stored in SQLite
- API deployments use per-app UUID tokens with constant-time comparison
- Middleware checks authentication based on endpoint type

**Execution Streaming:**
- ExecutorService maintains map of execution ID → subscriber channels
- Handlers subscribe via `Subscribe()` to receive real-time output
- Stdout/stderr streamed line-by-line to all subscribers
- Completion message broadcasted when execution finishes
- Subscribers cleanup via `Unsubscribe()`

**Database Migration Strategy:**
- Initial migrations run unconditionally (idempotent CREATE IF NOT EXISTS)
- Versioned migrations tracked in `migrations` table
- Each migration checks if already run before executing
- Complex schema changes use table recreation pattern (see `migrateExecutionsTable`)

**Configuration:**
- YAML-based config with sensible defaults
- Path prefix support for reverse proxy deployment
- Configurable timeouts, rate limits, session duration
- Auto-generated admin password if "changeme" detected

## Database Schema

Key tables and their relationships:

- **users**: Admin users for Web UI access
- **sessions**: Session tokens linked to users (CASCADE delete)
- **apps**: Applications with UUID, name, working_dir, and unique token
- **commands**: Shell commands linked to apps (CASCADE delete)
- **executions**: Command execution history, can be triggered by users OR API (nullable user_id)
- **audit_logs**: Security audit trail with user, action, resource tracking

Critical indexes on `executions(command_id, user_id, status)` for performance.

## Important Patterns & Conventions

### When Adding New Migrations

1. Create migration function in [migrations.go](internal/database/migrations.go)
2. Add unique migration name with timestamp prefix (e.g., `2025_12_06_000001_description`)
3. Check if migration already ran using `hasMigrationRun()`
4. Execute migration and record via `recordMigration()`
5. Add to `runVersionedMigrations()` function
6. For schema changes, use table recreation pattern if foreign keys involved

### When Adding New Routes

1. Define handler method in appropriate handler struct
2. Add route in [router.go](internal/router/router.go) under correct group (api/web/public)
3. Apply appropriate rate limiter middleware
4. Use session auth middleware for Web UI, token validation for deploy endpoints
5. Add audit logging in handler for sensitive operations

### When Adding New Command Execution Features

- All execution happens in `ExecutorService.Execute()`
- Use context.WithTimeout from command's TimeoutSeconds (capped by MaxTimeout)
- Always broadcast completion message to subscribers
- Update execution status in database at each phase (pending → running → success/failed)
- Log execution lifecycle events for debugging

### Error Handling

- Service layer returns errors, handlers convert to HTTP responses
- Database errors logged and returned as 500
- Validation errors returned as 400 with descriptive messages
- Authentication failures as 401, authorization failures as 403
- Use custom errors like `ErrExecutionNotFound` for clear error semantics

## Configuration

Default config path: `config.yaml` in working directory. Override with `-config` flag.

Key settings:
- `server.path_prefix`: URL prefix for reverse proxy (e.g., `/devops`)
- `server.secure_cookie`: Enable for HTTPS in production
- `database.path`: SQLite file location
- `execution.max_timeout`: Hard limit on command execution time
- `admin.password`: If "changeme", auto-generates secure password on first run

## Security Considerations

- Command execution runs as the process user (typically root in systemd service)
- Working directory validation checks existence before execution
- Token comparison uses constant-time to prevent timing attacks
- Rate limiting prevents brute-force on login and API abuse
- Session cookies use HttpOnly flag, Secure flag when configured
- Bcrypt cost factor 12 for password hashing
- Audit logs track all sensitive operations with IP and user agent

## Testing Notes

- Use `testing.Short()` flag for skipping slow tests
- Database tests use in-memory SQLite (`:memory:`)
- Handler tests create test Gin engine with test database
- Service tests mock database where appropriate
- Integration tests in `*_integration_test.go` files
- CI runs with race detector and generates coverage reports

## External Dependencies

- `github.com/gin-gonic/gin`: HTTP framework
- `github.com/mattn/go-sqlite3`: SQLite driver (requires CGO)
- `github.com/google/uuid`: UUID generation
- `golang.org/x/crypto`: bcrypt password hashing
- `gopkg.in/yaml.v3`: YAML config parsing

## Common Development Workflows

**Adding a new app management feature:**
1. Add model fields in [internal/models/app.go](internal/models/app.go:1-26)
2. Update database schema via migration
3. Add service method in AppService
4. Add handler method in AppHandler
5. Add route in router
6. Update Web UI templates if needed

**Adding a new authentication method:**
1. Add middleware in [internal/middleware/auth.go](internal/middleware/auth.go)
2. Update AuthService for token/credential validation
3. Update router to apply middleware to protected routes
4. Add audit logging for auth events

**Debugging execution issues:**
- Check logs for `[Executor]` prefixed messages
- Verify working directory exists and has correct permissions
- Check timeout settings in command vs. config
- Inspect `executions` table for status, output, exit_code
- Use SSE stream endpoint to see real-time output
