// Package swagger provides an HTTP handler to serve Swagger UI with embedded OpenAPI specs.
package swagger

import (
	"archive/zip"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// Resource represents a named OpenAPI spec URL for the Swagger UI dropdown.
type Resource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Spec is a single named OpenAPI spec entry. Specs are listed (and served
// in the Swagger UI dropdown) in the order they appear in Config.Specs.
type Spec struct {
	// Name is the display name and query-param identifier (e.g. "@staff (v3)").
	Name string
	// Data is the raw spec content (YAML or JSON bytes).
	Data []byte
}

// Config holds the configuration for the Swagger UI handler.
type Config struct {
	// BasePath is the URL path prefix where Swagger UI is served (e.g., "/swagger")
	BasePath string
	// SpecPath is the URL path to serve the OpenAPI spec (e.g., "/openapi/specs")
	SpecPath string
	// ResourcesPath is the URL path to serve the resources list (e.g., "/openapi/resources")
	ResourcesPath string
	// Specs is the ordered list of spec name/content pairs. The Swagger UI
	// dropdown lists them in this order.
	Specs []Spec
	// DefaultSpec is the name of the spec the UI opens on, and the one served
	// when no query param is provided
	DefaultSpec string
	// BundledInitializer serves the swagger-initializer.js inside the zip
	// instead of the one the handler writes from this Config. Set it only for
	// a zip built with an initializer of its own: the written one is what
	// keeps SpecPath, the spec list and DefaultSpec in step with the UI.
	BundledInitializer bool
}

// Handler serves Swagger UI and OpenAPI specifications.
type Handler struct {
	config        Config
	swaggerUI     fs.FS
	resourcesJSON []byte
	initializer   []byte
	specsByName   map[string][]byte
}

// initializerFile is the file Swagger UI's index.html loads to start itself.
const initializerFile = "swagger-initializer.js"

// New creates a new Swagger UI handler with the given configuration.
// swaggerUIZip should be the bytes of the swagger-ui.zip file.
func New(swaggerUIZip []byte, config Config) (*Handler, error) {
	if config.BasePath == "" {
		config.BasePath = "/swagger"
	}
	if config.SpecPath == "" {
		config.SpecPath = "/openapi/specs"
	}
	if config.ResourcesPath == "" {
		config.ResourcesPath = "/openapi/resources"
	}

	// Parse the zip file
	reader := bytes.NewReader(swaggerUIZip)
	zipReader, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, err
	}

	specsByName := make(map[string][]byte, len(config.Specs))
	resources := make([]Resource, 0, len(config.Specs))
	for _, s := range config.Specs {
		if _, dup := specsByName[s.Name]; dup {
			return nil, fmt.Errorf("swagger: spec %q is listed twice", s.Name)
		}
		specsByName[s.Name] = s.Data
		resources = append(resources, Resource{
			Name: s.Name,
			URL:  config.SpecPath + "?spec=" + url.QueryEscape(s.Name),
		})
	}

	resourcesJSON, err := json.Marshal(resources, jsontext.EscapeForHTML(true))
	if err != nil {
		return nil, err
	}

	initializer, err := writeInitializer(resourcesJSON, config)
	if err != nil {
		return nil, err
	}

	return &Handler{
		config:        config,
		swaggerUI:     zipReader,
		resourcesJSON: resourcesJSON,
		initializer:   initializer,
		specsByName:   specsByName,
	}, nil
}

// writeInitializer writes the script that starts Swagger UI over the specs this
// handler serves. Every value is JSON-encoded with <, > and & escaped, so a
// spec name cannot end the script.
//
// It lists the specs itself rather than fetching ResourcesPath, and reads
// nothing from the page URL: an initializer that took the spec location from a
// query parameter would render a spec from anywhere under this origin.
func writeInitializer(resourcesJSON []byte, config Config) ([]byte, error) {
	primary, err := json.Marshal(config.DefaultSpec, jsontext.EscapeForHTML(true))
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, `window.onload = function () {
  window.ui = SwaggerUIBundle({
    urls: %s,
    "urls.primaryName": %s,
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    plugins: [SwaggerUIBundle.plugins.DownloadUrl],
    layout: "StandaloneLayout",
    docExpansion: "none",
    deepLinking: true,
    operationsSorter: "alpha",
    tagsSorter: function (a, b) {
      if (a === "Authentication") return -1;
      if (b === "Authentication") return 1;
      return a.localeCompare(b);
    },
  });
};
`, resourcesJSON, primary), nil
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Redirect /swagger to /swagger/ for proper relative path resolution
	if r.URL.Path == h.config.BasePath {
		http.Redirect(w, r, h.config.BasePath+"/", http.StatusMovedPermanently)
		return
	}

	switch {
	case strings.HasPrefix(r.URL.Path, h.config.BasePath+"/"):
		h.serveSwaggerUI(w, r)
	case r.URL.Path == h.config.SpecPath:
		h.serveSpec(w, r)
	case r.URL.Path == h.config.ResourcesPath:
		h.serveResources(w, r)
	default:
		http.NotFound(w, r)
	}
}

// ServeUI serves Swagger UI files. Use this when the router strips the base path
// (e.g., chi.Mount). The path should be relative to the mount point.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	// Redirect root to / for proper relative path resolution
	if r.URL.Path == "" || r.URL.Path == "/" {
		// Check if we're at the mount point without trailing slash
		// by looking at the original URL
		if !strings.HasSuffix(r.RequestURI, "/") {
			// Use a relative redirect ("./") so the browser appends a trailing
			// slash to the current URL. Avoids passing attacker-controlled
			// r.RequestURI to http.Redirect (gosec G710 open redirect).
			http.Redirect(w, r, "./", http.StatusMovedPermanently)
			return
		}
	}

	h.serveFile(w, r, strings.TrimPrefix(r.URL.Path, "/"))
}

// Routes registers the handler routes on the given mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.Handle(h.config.BasePath+"/", h)
	mux.HandleFunc(h.config.SpecPath, h.serveSpec)
	mux.HandleFunc(h.config.ResourcesPath, h.serveResources)
}

// ServeSpec serves the OpenAPI spec. Use with chi: router.Get("/openapi/specs", h.ServeSpec)
func (h *Handler) ServeSpec(w http.ResponseWriter, r *http.Request) {
	h.serveSpec(w, r)
}

// ServeResources serves the resources list. Use with chi: router.Get("/openapi/resources", h.ServeResources)
func (h *Handler) ServeResources(w http.ResponseWriter, r *http.Request) {
	h.serveResources(w, r)
}

func (h *Handler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// Strip the base path prefix
	filePath := strings.TrimPrefix(r.URL.Path, h.config.BasePath)
	h.serveFile(w, r, strings.TrimPrefix(filePath, "/"))
}

// serveFile serves one file of the Swagger UI bundle. http.ServeFileFS sets the
// content type, answers HEAD, ranges and conditional requests from the zip
// entry's modification time, and refuses a path that leaves the bundle.
func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, filePath string) {
	if filePath == "" {
		filePath = "index.html"
	}
	if filePath == initializerFile && !h.config.BundledInitializer {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(h.initializer)
		return
	}
	if !fs.ValidPath(filePath) {
		http.NotFound(w, r)
		return
	}
	if info, err := fs.Stat(h.swaggerUI, filePath); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFileFS(w, r, h.swaggerUI, filePath) //nolint:gosec // G703: a valid path inside the embedded zip, checked above
}

func (h *Handler) serveSpec(w http.ResponseWriter, r *http.Request) {
	specName := r.URL.Query().Get("spec")
	if specName == "" {
		specName = h.config.DefaultSpec
	}

	spec, ok := h.specsByName[specName]
	if !ok {
		// If no specific spec requested and we have a default, use the first one.
		if specName == "" && len(h.config.Specs) > 0 {
			spec = h.config.Specs[0].Data
		} else {
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", specContentType(spec))
	_, _ = w.Write(spec)
}

// specContentType tells a JSON document from a YAML one by its first
// significant byte.
func specContentType(spec []byte) string {
	if trimmed := bytes.TrimSpace(spec); len(trimmed) > 0 && trimmed[0] == '{' {
		return "application/json"
	}
	return "application/yaml"
}

func (h *Handler) serveResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.resourcesJSON)
}
