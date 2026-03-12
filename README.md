# apikit

[![Tests](https://github.com/kausys/apikit/actions/workflows/test.yml/badge.svg)](https://github.com/kausys/apikit/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kausys/apikit)](https://goreportcard.com/report/github.com/kausys/apikit)

Go toolkit for HTTP handler generation, OpenAPI 3.1 specification generation, and Go SDK generation.

## Tools

- **`handler gen`** — Generate HTTP handler wrappers from annotated Go functions
- **`openapi gen`** — Generate OpenAPI 3.1 specs from Go source code (swagger directives)
- **`sdk gen`** — Generate typed Go SDK packages from OpenAPI specs
- **`swagger`** — Download and package Swagger UI assets

## Installation

```bash
go install github.com/kausys/apikit/cmd/apikit@latest
```

---

## 1. handler gen — HTTP Handler Generation

Generates request parsing, validation, and response handling boilerplate from annotated Go functions.

### Quick Start

```go
//go:generate apikit handler gen

type GetUserRequest struct {
    UserID string `path:"id" validate:"required,uuid"`
    Fields string `query:"fields"`
}

type GetUserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
}

// apikit:handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{ID: req.UserID, Name: "John"}, nil
}
```

```bash
go generate ./...
```

```go
// Standard http
mux.HandleFunc("GET /users/{id}", getUserAPIKit(GetUser))

// Fiber
app.Get("/users/:id", getUserAPIKit(GetUser))

// Gin
router.GET("/users/:id", getUserAPIKit(GetUser))

// Echo
e.GET("/users/:id", getUserAPIKit(GetUser))
```

### Parameter Sources

| Tag | Source |
|-----|--------|
| `path:"name"` | URL path parameter |
| `query:"name"` | Query string |
| `header:"name"` | HTTP header |
| `cookie:"name"` | Cookie |
| `form:"name"` | Form field / multipart |
| `json:"name"` | JSON body |

### Validation

Uses [go-playground/validator](https://github.com/go-playground/validator):

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
}
```

### File Uploads

```go
type UploadRequest struct {
    File  *multipart.FileHeader   `form:"file"`
    Files []*multipart.FileHeader `form:"files"`
}
```

### Raw HTTP Access

```go
type StreamRequest struct {
    Request *http.Request
    Writer  http.ResponseWriter
    RawBody []byte
}
```

### CLI Reference

```
apikit handler gen [flags]

Flags:
  -f, --file string        Source file to process (defaults to $GOFILE)
  -o, --output string      Output file (defaults to <source>_apikit.go)
      --framework string   Target framework: http, fiber, gin, echo (default "http")
      --force              Force regeneration even if source is unchanged
      --dry-run            Print output without writing
  -v, --verbose            Verbose output
```

---

## 2. openapi gen — OpenAPI 3.1 Specification Generation

Scans Go source code for swagger directives and generates a complete OpenAPI 3.1 spec.

### Supported Directives

| Directive | Purpose |
|-----------|---------|
| `swagger:meta` | API metadata (title, version, description) |
| `swagger:model` | Schema definitions |
| `swagger:route` | Operation definitions |
| `swagger:parameters` | Parameter definitions |
| `swagger:enum` | Enum definitions |

### Examples

```bash
# Generate to openapi.yaml (default)
apikit openapi gen

# Custom output and format
apikit openapi gen -o api.json -f json

# Scan a specific package pattern
apikit openapi gen -p ./api/... -o api.yaml

# Generate multiple specs from spec: directives
apikit openapi gen --multi-specs

# Generate a single named spec
apikit openapi gen --spec public

# Inline $ref schemas
apikit openapi gen --flatten

# Validate the generated spec
apikit openapi gen --validate

# Clean cache and regenerate
apikit openapi clean && apikit openapi gen
```

### CLI Reference

```
apikit openapi gen [flags]

Flags:
  -o, --output string       Output file path (default "openapi.yaml")
  -f, --format string       Output format: yaml or json (default "yaml")
  -p, --pattern string      Package pattern to scan (default "./...")
  -d, --dir string          Root directory to scan from (default ".")
      --no-cache            Disable incremental caching
      --flatten             Inline $ref schemas instead of using references
      --validate            Validate the generated spec
      --ignore strings      Path patterns to ignore
      --clean-unused        Remove unreferenced schemas
      --multi-specs         Generate multiple specs based on spec: directives
      --spec string         Generate only a specific spec by name
      --no-default          Skip generating the default spec for untagged routes
      --enum-refs           Generate enums as $ref instead of inline

apikit openapi clean        Remove the .openapi cache directory
apikit openapi status       Show cache statistics
```

---

## 3. sdk gen — Go SDK Generation

Generates a complete, typed Go SDK package from an OpenAPI specification and a `.sdkgen.yaml` config.

### Generated Structure

```
pkg/sdk/pokemon/
├── client/    # HTTP client with middleware chain
├── config/    # Configuration (gookit/config)
├── models/    # Request/response structs and enums
└── services/  # Service methods per API tag
```

### Examples

```bash
# Generate SDK to ./pkg/sdk/pokemon
apikit sdk gen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon

# Override provider name
apikit sdk gen pokemon.sdkgen.yaml -o ./pkg/sdk/pokemon --provider myProvider
```

### CLI Reference

```
apikit sdk gen <config.sdkgen.yaml> [flags]

Args:
  config.sdkgen.yaml   Path to SDK config file (required)

Flags:
  -o, --output string     Output directory for generated SDK (required)
      --provider string   Override provider name from config
```

---

## 4. swagger — Swagger UI Management

Downloads and packages Swagger UI assets for embedding in Go applications.

### Examples

```bash
# Download latest Swagger UI
apikit swagger download -o ./pkg/openapi

# Download specific version
apikit swagger download -v 5.29.4 -o ./pkg/openapi

# Download without default customizations
apikit swagger download --with-defaults=false -o ./pkg/openapi

# Single-spec mode initializer
apikit swagger download --simple -o ./pkg/openapi

# Check latest available version
apikit swagger version
```

### Using the Downloaded Assets

```go
//go:embed swagger-ui.zip
var swaggerUIData []byte

handler, err := swagger.New(swaggerUIData, swagger.Config{
    Specs: map[string][]byte{"api": specData},
})
```

### CLI Reference

```
apikit swagger download [flags]

Flags:
  -o, --output string    Output directory for swagger-ui.zip (default ".")
  -v, --version string   Specific version to download (default: latest)
      --with-defaults    Include default initializer and CSS (default true)
      --simple           Use simple initializer for single-spec mode

apikit swagger version   Show the latest available Swagger UI version
```

---

## Workspace Architecture

The repository is a Go workspace (`go.work`) with 5 independent modules:

| Module | Import Path | Purpose |
|--------|-------------|---------|
| `runtime/` | `github.com/kausys/apikit/runtime` | HTTP utilities, error types, validation |
| `scanner/` | `github.com/kausys/apikit/scanner` | Go AST scanner for swagger directives |
| `openapi/` | `github.com/kausys/apikit/openapi` | OpenAPI spec types, generator, SDK gen, swagger |
| `handler/` | `github.com/kausys/apikit/handler` | Handler parser and code generator |
| `cmd/` | `github.com/kausys/apikit/cmd` | CLI (`apikit` binary) |

## Development

```bash
# Build all modules
make build

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Lint all modules
make lint

# Format all modules
make fmt

# Tidy all modules
make tidy

# Full CI check (fmt + lint + tidy check + test)
make ci

# Install apikit locally
make install

# Install dev tools
make setup
```

## License

MIT License — see [LICENSE](LICENSE) for details.
