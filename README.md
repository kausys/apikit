# APIKit

[![Tests](https://github.com/kausys/apikit/actions/workflows/test.yml/badge.svg)](https://github.com/kausys/apikit/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kausys/apikit)](https://goreportcard.com/report/github.com/kausys/apikit)
[![GoDoc](https://pkg.go.dev/badge/github.com/kausys/apikit)](https://pkg.go.dev/github.com/kausys/apikit)

Code generator for Go HTTP handlers. Automatically generates request parsing, validation, and response handling from annotated handler functions.

## Features

- 🚀 **Zero boilerplate** - Focus on business logic, not HTTP parsing
- 🔌 **Multi-framework** - Supports http, Fiber, Gin, and Echo
- ✅ **Validation** - Built-in struct validation with go-playground/validator
- 📝 **Type-safe** - Full type safety with generics
- ⚡ **go:generate** - Seamless integration with Go toolchain

## Installation

```bash
go install github.com/kausys/apikit/cmd/apikit@latest
```

## Quick Start

### 1. Define your handler

```go
//go:generate apikit generate

// GetUserRequest defines the parameters for GetUser
type GetUserRequest struct {
    UserID string `path:"id" validate:"required,uuid"`
    Fields string `query:"fields"`
}

type GetUserResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// apikit:handler
func GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    // Your business logic here
    return GetUserResponse{ID: req.UserID, Name: "John"}, nil
}
```

### 2. Generate the wrapper

```bash
go generate ./...
```

### 3. Use with your framework

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

## Parameter Sources

| Tag | Source | Example |
|-----|--------|---------|
| `path:"name"` | URL path parameter | `/users/{id}` |
| `query:"name"` | Query string | `?filter=active` |
| `header:"name"` | HTTP header | `X-API-Key` |
| `cookie:"name"` | Cookie | `session_id` |
| `form:"name"` | Form field | `multipart/form-data` |
| `json:"name"` | JSON body | Request body |

## Validation

Uses [go-playground/validator](https://github.com/go-playground/validator):

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=120"`
}
```

## Framework Selection

```bash
# Standard http (default)
apikit generate

# Fiber
apikit generate --framework fiber

# Gin
apikit generate --framework gin

# Echo
apikit generate --framework echo
```

## Special Fields

Access raw HTTP primitives when needed:

```go
type StreamRequest struct {
    Request  *http.Request       // Access full request
    Writer   http.ResponseWriter // Access response writer
    RawBody  []byte              // Raw request body
}
```

## File Uploads

```go
type UploadRequest struct {
    File  *multipart.FileHeader   `form:"file"`
    Files []*multipart.FileHeader `form:"files"`
}
```

## CLI Reference

```bash
apikit generate [flags]

Flags:
  -f, --file string        Source file (defaults to $GOFILE)
  -o, --output string      Output file (defaults to <source>_apikit.go)
      --framework string   Target framework: http, fiber, gin, echo (default "http")
      --force              Force regeneration
      --dry-run            Show output without writing
  -v, --verbose            Verbose output
```

## License

MIT License - see [LICENSE](LICENSE) for details.

