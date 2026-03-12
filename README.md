# apikit

[![Tests](https://github.com/kausys/apikit/actions/workflows/test.yml/badge.svg)](https://github.com/kausys/apikit/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kausys/apikit)](https://goreportcard.com/report/github.com/kausys/apikit)

Go toolkit for building APIs: HTTP handler generation, OpenAPI 3.1 spec generation, Go SDK generation, and runtime utilities.

## Modules

| Module | Import Path | Description |
|--------|-------------|-------------|
| **root** | `github.com/kausys/apikit` | Runtime: HTTP responses, errors, validation, logging, sanitization |
| **scanner** | `github.com/kausys/apikit/scanner` | Go AST scanner for swagger directives |
| **openapi** | `github.com/kausys/apikit/openapi` | OpenAPI 3.1 spec generation from scanned directives |
| **handler** | `github.com/kausys/apikit/handler` | HTTP handler code generation from annotated functions |
| **cmd** | `github.com/kausys/apikit/cmd` | CLI binary (`apikit`) |

## Installation

```bash
# CLI tool
go install github.com/kausys/apikit/cmd/apikit@latest

# Runtime library (used by generated code)
go get github.com/kausys/apikit@latest
```

---

## Root Module — Runtime

`github.com/kausys/apikit`

The root module provides the runtime types and utilities that generated handler code depends on.

### HTTP Responses

```go
import "github.com/kausys/apikit"

// Return structured responses from handlers
response := apikit.NewHttpResponse(http.StatusOK, data)
response.WithHeader("X-Request-ID", reqID)
response.WithContentType("application/json")

// Write JSON directly
apikit.WriteJSON(w, data)

// Handle response/error from a handler call
apikit.HandleResponse(w, response, err)
```

### Errors

```go
// Predefined error constructors
apikit.BadRequest("invalid email format")
apikit.NotFound("user not found")
apikit.Unauthorized("invalid token")
apikit.Forbidden("insufficient permissions")
apikit.Conflict("email already exists")
apikit.UnprocessableEntity("validation failed")
apikit.InternalError("database connection failed")
apikit.TooManyRequests("rate limit exceeded")
apikit.ServiceUnavailable("service down")

// With details and chaining
apikit.BadRequest("validation failed").
    WithDetails(fieldErrors).
    WithRequestID(reqID).
    WithCause(err)
```

### Validation

```go
import "github.com/kausys/apikit/validator"

// Validate structs
err := validator.StructCtx(ctx, &payload)

// Register custom validations
validator.RegisterValidation(func(v *validator.Validate) {
    v.RegisterValidation("custom_rule", customFunc)
})

// Implement ValidEnum for enum types
type Status string
func (s Status) IsValid() bool { return s == "active" || s == "inactive" }
```

### Logging

```go
import "github.com/kausys/apikit"

// Set a global logger (slog-style key-value pattern)
apikit.SetLogger(myLogger)

// Check if a logger is configured
if apikit.LoggerEnabled() {
    // ...
}
```

### Sanitization

Sanitize structs for safe logging using struct tags:

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Password string `json:"password" log:"sensitive"` // → "[REDACTED]"
    Internal string `json:"-"        log:"-"`          // → omitted
}

// In log calls
logger.Info("request", "payload", apikit.Sanitize(payload))
// Output: {"email": "user@example.com", "password": "[REDACTED]"}
```

| Tag | Behavior |
|-----|----------|
| `log:"-"` | Field omitted from output |
| `log:"sensitive"` | Value replaced with `[REDACTED]` |
| *(none)* | Value passed through |

Supports nested structs, pointers, slices, and embedded structs. Uses `json` tag names as map keys.

### Swagger UI Handler

```go
import "github.com/kausys/apikit/swagger"

//go:embed swagger-ui.zip
var swaggerUIZip []byte

handler, err := swagger.New(swaggerUIZip, swagger.Config{
    BasePath:      "/swagger",
    SpecPath:      "/openapi/specs",
    ResourcesPath: "/openapi/resources",
    Specs: map[string][]byte{
        "Public API":   publicSpec,
        "Internal API": internalSpec,
    },
    DefaultSpec: "Public API",
})

// Option 1: Use with http.ServeMux
handler.Routes(mux)

// Option 2: Use with chi or similar routers
r.Mount("/swagger", http.StripPrefix("/swagger", http.HandlerFunc(handler.ServeUI)))
r.Get("/openapi/specs", handler.ServeSpec)
r.Get("/openapi/resources", handler.ServeResources)
```

### Time Parsing

```go
t, err := apikit.NewTimeFromString("2024-01-15T10:30:00Z")
// Supports: RFC3339, RFC3339Nano, 2006-01-02, 01/02/2006, etc.
```

---

## Scanner Module

`github.com/kausys/apikit/scanner`

Parses Go source files to extract swagger directives into structured data.

```go
import "github.com/kausys/apikit/scanner"

s := scanner.New(
    scanner.WithDir("."),
    scanner.WithPattern("./..."),
    scanner.WithIgnorePaths("vendor", "testdata"),
)

if err := s.Scan(); err != nil {
    log.Fatal(err)
}

// Access results
s.Meta       // *MetaInfo — API metadata
s.Structs    // map[string]*StructInfo — models and parameters
s.Routes     // map[string]*RouteInfo — API endpoints
s.Enums      // map[string]*EnumInfo — enum definitions
```

### Swagger Directives

```go
// swagger:meta
// Title: My API
// Version: 1.0.0
// Description: API description
// BasePath: /api/v1
// SecuritySchemes:
//   BearerAuth:
//     type: http
//     scheme: bearer

// swagger:model
type User struct {
    ID    string `json:"id"`
    Email string `json:"email" example:"user@example.com"`
}

// swagger:enum
type Status string
const (
    StatusActive   Status = "active"
    StatusInactive Status = "inactive"
)

// swagger:parameters
type GetUserParams struct {
    // in: path
    UserID string `json:"user_id" validate:"required,uuid"`
}

// swagger:route GET /users/{user_id} users GetUser
// summary: Get a user by ID
// Responses:
//   200: User
//   404: ErrorResponse
// Security:
//   BearerAuth
```

### Multi-Spec Support

Assign elements to specific specs using the `spec:` directive:

```go
// swagger:route GET /public/health public Health
// spec: public
// summary: Health check

// swagger:route GET /internal/metrics internal Metrics
// spec: internal
// summary: Internal metrics
```

---

## OpenAPI Module

`github.com/kausys/apikit/openapi`

Generates OpenAPI 3.1 specifications from scanner output.

```go
import "github.com/kausys/apikit/openapi"

doc, err := openapi.Generate(
    openapi.WithDir("."),
    openapi.WithPattern("./api/..."),
    openapi.WithOutput("openapi.yaml", "yaml"),
    openapi.WithCache(true),
    openapi.WithFlatten(false),
    openapi.WithValidation(true),
    openapi.WithCleanUnused(true),
    openapi.WithEnumRefs(true),
)
```

### Custom Type Mapping

Create `.openapi.yaml` in your project root to map Go types to OpenAPI types:

```yaml
custom_types:
  # Short name (backward compatible)
  decimal.Decimal:
    type: string
    format: decimal
    example: "123.45"

  # Fully-qualified path (avoids collisions)
  github.com/shopspring/decimal.Decimal:
    type: string
    format: decimal
    example: "123.45"
```

Or register programmatically:

```go
import "github.com/kausys/apikit/openapi/generator"

generator.FieldType(decimal.Decimal{}, func(info *generator.TypeInfo) {
    info.Type = "string"
    info.Format = "decimal"
    info.Example = "123.45"
})
```

### Subpackages

| Package | Purpose |
|---------|---------|
| `openapi/spec` | OpenAPI 3.1 type definitions (Schema, Paths, Components, etc.) |
| `openapi/generator` | Spec generation engine, custom type registry |
| `openapi/cache` | Incremental caching for large codebases |
| `openapi/sdkgen` | Go SDK generation from OpenAPI specs |
| `openapi/swagger` | Swagger UI asset management |

---

## Handler Module

`github.com/kausys/apikit/handler`

Generates HTTP handler wrappers with request parsing, validation, and response handling.

### Usage

```go
//go:generate apikit handler gen

type GetUserRequest struct {
    UserID string `path:"id" validate:"required,uuid"`
    Fields string `query:"fields"`
}

type GetUserResponse struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// apikit:handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    return GetUserResponse{ID: req.UserID, Name: "John"}, nil
}
```

```bash
go generate ./...
```

### Routing

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
| `cookie:"name"` | Cookie value |
| `form:"name"` | Form field / multipart |
| `json:"name"` | JSON body (auto-detected) |

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

### Subpackages

| Package | Purpose |
|---------|---------|
| `handler/parser` | Parses Go files for `apikit:handler` annotations |
| `handler/codegen` | Generates handler wrapper code from templates |
| `handler/extractors` | Framework-specific parameter extraction (http, fiber, gin, echo) |
| `handler/types` | Custom type extractor registry for parameter parsing |

---

## CLI Reference

### handler gen

```
apikit handler gen [flags]

Flags:
  -f, --file string        Source file (defaults to $GOFILE)
  -o, --output string      Output file (defaults to <source>_apikit.go)
      --framework string   http, fiber, gin, echo (default "http")
      --force              Regenerate even if unchanged
      --dry-run            Print without writing
  -v, --verbose            Verbose output
```

### openapi gen

```
apikit openapi gen [flags]

Flags:
  -o, --output string       Output file (default "openapi.yaml")
  -f, --format string       yaml or json (default "yaml")
  -p, --pattern string      Package pattern (default "./...")
  -d, --dir string          Root directory (default ".")
      --no-cache            Disable caching
      --flatten             Inline $ref schemas
      --validate            Validate output
      --ignore strings      Paths to ignore
      --clean-unused        Remove unreferenced schemas
      --multi-specs         Generate multiple specs
      --spec string         Generate specific spec only
      --no-default          Skip default spec
      --enum-refs           Enums as $ref

apikit openapi clean        Remove cache
apikit openapi status       Show cache stats
```

### sdk gen

```
apikit sdk gen <config.sdkgen.yaml> -o <output-dir> [flags]

Flags:
  -o, --output string     Output directory (required)
      --provider string   Override provider name
```

### swagger

```
apikit swagger download [flags]

Flags:
  -o, --output string    Output directory (default ".")
  -v, --version string   Version (default: latest)
      --with-defaults    Include default initializer (default true)
      --simple           Single-spec mode

apikit swagger version   Show latest version
```

---

## Development

```bash
make setup          # Install golangci-lint v2 + lefthook
make build          # Build all modules
make test           # Run all tests
make test-coverage  # Tests with coverage
make lint           # Lint all modules
make lint-fix       # Auto-fix lint issues
make fmt            # Format all modules
make tidy           # go mod tidy all modules
make ci             # Full CI check (fmt + lint + tidy + test)
make install        # Install apikit binary locally
```

## License

MIT License — see [LICENSE](LICENSE) for details.
