# apikit

[![Tests](https://github.com/kausys/apikit/actions/workflows/test.yml/badge.svg)](https://github.com/kausys/apikit/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kausys/apikit)](https://goreportcard.com/report/github.com/kausys/apikit)

Go toolkit for building APIs: HTTP handler generation, OpenAPI 3.1 spec generation, Go SDK generation, and the runtime the generated code uses.

## Modules

| Module | Import path | What |
|--------|-------------|------|
| **root** | `github.com/kausys/apikit` | Runtime: responses, errors, request bodies, validation, logging, sanitization, Swagger UI serving |
| **cmd** | `github.com/kausys/apikit/cmd` | The `apikit` CLI and its generators (`handler`, `openapi`, `scanner`, `authz`, `sdkgen`) |

Both are released together, at the same version. Require them at the same version too: generated wrappers call the runtime of their own release.

## Installation

```bash
# As a tool of your module (recommended: the version is pinned in go.mod)
go get -tool github.com/kausys/apikit/cmd/apikit@latest
go tool apikit --help

# Or as a binary
go install github.com/kausys/apikit/cmd/apikit@latest

# The runtime the generated code imports
go get github.com/kausys/apikit@latest
```

Requires Go 1.27.

---

## Runtime

`github.com/kausys/apikit`

### JSON

Every body is read and written with `encoding/json/v2`:

- **Reading** (`DecodeJSON`, used by generated wrappers): member names match field names exactly — `{"Email": …}` does not fill `json:"email"` —, an object that names a member twice is refused, and invalid UTF-8 is refused. Unknown members are ignored.
- **Writing**: a nil slice or map is `[]` or `{}`, not `null`; `omitempty` omits empty JSON values (`""`, `null`, `[]`, `{}`), so use `omitzero` to omit a zero number or `false`; strings are not HTML-escaped; map keys are sorted.

### Responses

```go
// A body with a status, headers and cookies
return apikit.NewHttpResponse(http.StatusCreated, user).
    WithHeader("Location", "/users/"+user.ID).
    WithCookie(sessionCookie), nil
```

A handler that returns any other value answers 200 with it as JSON. The body is encoded before anything is written: a value that cannot be encoded answers 500, not the intended status over a truncated body. `204` and `304` carry no body.

### Errors

```go
apikit.BadRequest("invalid email format")
apikit.NotFound("user")                 // "user not found"
apikit.Unauthorized("invalid token")
apikit.Conflict("email already exists")
apikit.UnprocessableEntity("validation failed")
apikit.PayloadTooLarge("request body too large")
apikit.InternalError("could not save")

apikit.BadRequest("validation failed").
    WithDetails(fieldErrors).
    WithRequestID(reqID).
    WithCause(err)
```

- The status comes from any `*apikit.Error` (or anything with `StatusCode() int`) in the error chain, wrapped or not.
- Any other error answers 500 with `"internal server error"`; what failed goes to the logger, not to the client.
- The predefined errors (`ErrUnauthorized`, `ErrForbidden`, …) are shared by every request: a `With*` call on one returns a copy, and `errors.Is(copy, apikit.ErrUnauthorized)` still holds.
- Replace the error format with `apikit.SetErrorRenderer` — an RFC 7807 renderer, for instance. `apikit.FieldErrorsOf(err)` gives a renderer the validation failures.

### Request bodies

```go
apikit.SetMaxBodyBytes(1 << 20)       // default 10 MiB
apikit.SetMaxMultipartBytes(32 << 20) // default 64 MiB
```

A body over the limit is refused with 413. It is never cut short: a truncated JSON body used to be reported as malformed, and a truncated raw body reached the handler as if it were whole.

### Logging

```go
apikit.SetLogger(apikit.NewSlogLogger(slog.Default()))
```

Generated wrappers report a request they could not serve through `apikit.LogFailure`, with the handler, method, path and error — never the payload. The level follows whose fault it was: an error that renders as 4xx goes to `Debug` when the logger implements `DebugLogger`, and nowhere otherwise; anything else goes to `Error`.

### Validation

```go
import "github.com/kausys/apikit/validator"

err := validator.StructCtx(ctx, &payload)

validator.RegisterValidation(func(v *validator.Validate, tr ut.Translator) {
    _ = v.RegisterValidation("slug", validSlug)
})

// Enums: implement IsValid (and optionally Values, for the message)
type Status string
func (s Status) IsValid() bool { return s == "active" || s == "inactive" }
```

Field names in validation errors are the JSON names, with nested paths (`address.city`).

### Sanitization

```go
type CreateUserRequest struct {
    Email    string `json:"email"`
    Password string `json:"password" log:"sensitive"` // "[REDACTED]"
    Internal string `json:"-"`                         // omitted
}

logger.Info("request", "payload", apikit.Sanitize(payload))
```

| Tag | Behavior |
|-----|----------|
| `log:"-"` | Field omitted |
| `log:"sensitive"` | Value replaced with `[REDACTED]` |
| `json:"-"` | Field omitted — kept off the wire, kept out of the log |
| *(none)* | Value passed through |

Structs are walked wherever they are: behind pointers and interfaces, in slices, arrays and map values. A type that renders itself (`time.Time`, a JSON or text marshaler, a `fmt.Stringer`, a `slog.LogValuer`) is kept as it is. Anything deeper than ten levels is `[TRUNCATED]`.

### Swagger UI

```go
import "github.com/kausys/apikit/swagger"

//go:embed swagger-ui.zip
var swaggerUI []byte

//go:embed specs/public.yaml
var publicSpec []byte

h, err := swagger.New(swaggerUI, swagger.Config{
    Specs: []swagger.Spec{
        {Name: "Public API", Data: publicSpec},
    },
    DefaultSpec: "Public API",
})

// net/http
h.Routes(mux)

// chi
r.Mount("/swagger", h)
r.Get("/openapi/specs", h.ServeSpec)
r.Get("/openapi/resources", h.ServeResources)
```

The handler writes `swagger-initializer.js` from the `Config` — the spec list, `SpecPath` and the default spec — so the UI always asks for the specs where they are served. It takes nothing from the page URL. Set `BundledInitializer` to serve the one inside the zip instead.

---

## Handler generation

`apikit handler gen` writes, next to each file with annotated handlers, a `<file>_apikit.go` with one wrapper per handler: it binds the request, validates it, calls the handler and writes the response or the error.

```go
//go:generate go tool apikit handler gen

type UpdateUserRequest struct {
    UserID UserID         `path:"user_id" validate:"required"`
    Fields []string       `query:"field"`
    Limit  *int           `query:"limit" default:"20"`
    Trace  string         `header:"X-Trace-Id"`
    Body   UpdateUserBody `json:"body"`
}

type UpdateUserBody struct {
    Name string `json:"name" validate:"required,min=2"`
}

// apikit:handler
func (h *Handler) UpdateUser(ctx context.Context, req UpdateUserRequest) (User, error)
```

```go
mux.Handle("PATCH /users/{user_id}", updateUserAPIKit(h.UpdateUser))
```

### Binding

| Tag | Source |
|-----|--------|
| `path:"name"` | Path parameter (`r.PathValue`) |
| `query:"name"` | Query string; a slice takes every value |
| `header:"Name"` | Header; a slice takes every value |
| `cookie:"name"` | Cookie |
| `form:"name"` | Form field or file (`*multipart.FileHeader`) |
| `json:"body"` | The JSON body is decoded into this field |
| `default:"value"` | Used when the request leaves the value out, parsed like a sent value |

- The body is decoded into the field named `body` (or marked `// in: body`) and nowhere else. A payload field with any other json name and no binding is refused at generation: it would never be filled. (A handler that also takes the `*http.Request` may read the body itself.)
- A field type may be any scalar, a pointer to one, a slice, `time.Time`, `uuid.UUID`, a type registered in `types`, or any type implementing `encoding.TextUnmarshaler`; any other named type is converted from the string.
- The payload, and every struct it uses, may be declared in any file of the handler's package, or in another package of the module.
- Validation runs when any field at any depth has a `validate` tag.

### Regeneration

A wrapper records a checksum of what it was generated from — the generator's format, the framework, and the packages its types come from — and is skipped while that is unchanged. A file that loses its last handler loses its wrapper, and a wrapper whose source file is gone is removed.

```bash
go tool apikit handler gen ./...           # write what changed
go tool apikit handler gen --check ./...   # CI: fail on a missing, stale or orphaned wrapper
```

---

## OpenAPI generation

`apikit openapi gen` scans for swagger directives and writes OpenAPI 3.1.

```go
// swagger:meta
//
// Title: Users API
// Version: 1.0.0
//
// SecuritySchemes:
//   - name: oauth
//     type: http
//     scheme: bearer

// UpdateUser changes a user.
//
// swagger:route PATCH /users/{user_id} users updateUser
//
// Security:
//   - oauth: users.write
//
// Responses:
//   200: User
//   404: ErrorResponse
//
// apikit:handler
func (h *Handler) UpdateUser(ctx context.Context, req UpdateUserRequest) (User, error)
```

### Parameters from handlers

A route declared on a handler needs no `swagger:parameters` struct: its parameters and body are read from the handler's request type, with the binding tags above. What is documented is what the wrapper parses. The body's struct, and the structs it refers to, become schemas without a `swagger:model` directive. A `swagger:parameters <operationId>` struct still takes precedence.

### Validation rules

The rules of a `validate` tag become schema keywords for the kind of value they constrain:

| Rule | string | array | object | number |
|------|--------|-------|--------|--------|
| `min`, `gte` | `minLength` | `minItems` | `minProperties` | `minimum` |
| `max`, `lte` | `maxLength` | `maxItems` | `maxProperties` | `maximum` |
| `gt`, `lt` | length ± 1 | count ± 1 | count ± 1 | `exclusiveMinimum`, `exclusiveMaximum` |
| `len` | both lengths | both counts | both counts | — |

`oneof` becomes `enum`; `email`, `uuid`, `url`, `uri`, `ipv4`, `ipv6`, `hostname`, `datetime` become formats; `unique` becomes `uniqueItems`; rules after `dive` apply to the items.

### Strictness

- `--validate` fails when the document is invalid: an operation without responses, a path parameter the template does not name (or the reverse), a parameter declared twice, a `$ref` or security scheme that names nothing.
- `--strict` also fails on everything that would silently degrade the spec: a package that does not type-check, a malformed `swagger:route`, a route without `spec:` under `--no-default`, and a declared response type that resolves to nothing.
- `--check` writes nothing and fails when a spec on disk is not what the source generates.

Specs are written only when they change. `--ignore` patterns match whole path segments of each package's import path and of each file path relative to `--dir`.

### Custom types

`.openapi.yaml` in the scanned directory:

```yaml
customTypes:
  decimal.Decimal:
    type: string
    format: decimal
    example: "123.45"
```

### A spec folder per binary

```
cmd/myapi/openapi/
├── doc.go           // swagger:meta and the go:generate directives
├── swagger.go       // //go:embed swagger-ui.zip and specs/*.yaml; swagger.New(...)
├── swagger-ui.zip   // apikit swagger download
└── specs/
    ├── public.yaml
    └── admin.yaml
```

```go
//go:generate go tool apikit swagger download
//go:generate go tool apikit openapi gen -d ../../.. -p ./... --multi-specs --no-default --clean-unused --strict -o ./specs/
```

`apikit swagger download` fetches the Swagger UI release this apikit pins and checks it against the checksum it knows, so the zip is the same on every machine. `--version latest` asks for the newest release; `--sha256` pins any other.

---

## CLI reference

```
apikit handler gen [files | dirs | dir/...] [flags]
  -f, --file string        Source file (defaults to $GOFILE)
  -o, --output string      Output file (one source only)
      --framework string   http, fiber, gin, echo (default "http")
      --force              Regenerate even if the inputs are unchanged
      --check              Write nothing; fail on a missing, stale or orphaned wrapper

apikit openapi gen [flags]
  -o, --output string      Output file, or the directory for --multi-specs (default "openapi.yaml")
  -f, --format string      yaml or json (default "yaml")
  -p, --pattern string     Package pattern (default "./...")
  -d, --dir string         Root directory (default ".")
      --ignore strings     Path patterns to ignore
      --clean-unused       Remove unreferenced schemas
      --multi-specs        One spec per spec: directive
      --spec string        Generate one spec only
      --no-default         Skip the spec of routes without spec:
      --enum-refs          Enums as $ref
      --validate           Fail on an invalid document
      --strict             Fail on any silent loss (implies --validate)
      --check              Write nothing; fail on a stale spec
apikit openapi clean       Remove the .openapi directory older versions left

apikit swagger download [flags]
  -o, --output string      Output directory (default ".")
      --version string     A release, or "latest" (default: the pinned one)
      --sha256 string      Checksum the release archive must have
      --with-defaults      Include the default initializer and CSS (default true)
      --simple             Single-spec initializer
apikit swagger version     Show the latest release

apikit sdk gen <config.sdkgen.yaml> -o <dir>
apikit authz gen ...

Global: --dry-run, -v/--verbose
```

---

## Development

```bash
make setup          # golangci-lint (pinned), lefthook, and the local go.work
make build          # Build both modules
make test           # Tests, with -race
make lint           # Lint both modules
make tidy           # go mod tidy, each module as its users see it
make ci             # fmt + lint + check-tidy + test
make install        # Install the apikit binary
```

`cmd` requires the last release of the root module. The local `go.work` (`make work`) builds it against the root module in the tree instead; it is not committed, because a `go.work` inside a module zip breaks its users.

## License

MIT License — see [LICENSE](LICENSE) for details.
