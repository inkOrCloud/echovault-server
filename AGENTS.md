# Repository Guidelines

## Project Structure & Module Organization

```
echovault-server/
├── cmd/server/         # Application entry point
├── internal/           # Internal packages (not exported)
│   ├── ent/           # Generated Ent ORM code
│   ├── grpc/          # gRPC handlers
│   ├── rest/          # REST API handlers
│   └── service/       # Business logic services
├── pkg/               # Public reusable packages
│   ├── auth/          # Authentication utilities
│   ├── config/        # Configuration management
│   ├── convert/       # Data conversion helpers
│   ├── metadata/      # Metadata parsing
│   └── storage/       # Storage implementations
├── api/grpc/          # Protobuf definitions
├── go.mod             # Go module definition
├── Makefile           # Build automation
├── Dockerfile         # Container build
└── .golangci.yml      # Linter configuration
```

## Build, Test, and Development Commands

| Command | Description |
|---------|-------------|
| `make build` | Compile the server binary to `bin/server` |
| `make run` | Start the development server |
| `make test` | Run all tests with race detection |
| `make generate` | Regenerate Ent ORM code after schema changes |
| `make proto-gen` | Generate gRPC code from protobuf definitions |
| `make clean` | Remove build artifacts |

## Coding Style & Naming Conventions

- **Formatter**: Follow `golangci-lint` rules configured in `.golangci.yml`
- **Linting**: Run `golangci-lint run ./...` before committing
- **Imports**: Group and sort imports per `gci` formatting rules
- **Errors**: Use plain error assignment (no inline error handling in function calls)
- **Naming**: Follow Go conventions—PascalCase for exported, camelCase for unexported
- **Error wrapping**: Wrap errors with context for debugging

## Testing Guidelines

- **Framework**: Standard Go testing with `go test`
- **Location**: Tests co-located with source files (e.g., `*_test.go`)
- **Run tests**: `make test` or `go test ./... -v -race -count=1`
- **Coverage**: Aim for reasonable coverage; focus on critical paths
- **Pattern**: Table-driven tests preferred for comprehensive scenarios

## Commit & Pull Request Guidelines

- **Commit messages**: Use conventional format: `<type>: <description>`
  - Types: `fix`, `feat`, `refactor`, `style`, `chore`, `docs`
  - Examples: `fix: resolve typecheck error in storage`, `style: reorder imports`
- **Pull requests**: Include clear description of changes and link related issues
- **Pre-commit**: Run `golangci-lint run ./...` to catch issues early
- **Branch naming**: Use descriptive names like `feature/sync-v2` or `fix/auth-error`

## Architecture Notes

- **Dual API**: Supports both gRPC and REST interfaces
- **ORM**: Uses Ent for type-safe database operations
- **Services**: Business logic in `internal/service/` with clean separation
- **Testing**: Comprehensive test coverage across handlers and services
