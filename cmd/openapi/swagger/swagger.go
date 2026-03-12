// Package swagger provides an HTTP handler to serve Swagger UI with embedded OpenAPI specs.
package swagger

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
)

// Resource represents a named OpenAPI spec URL for the Swagger UI dropdown.
type Resource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Config holds the configuration for the Swagger UI handler.
type Config struct {
	// BasePath is the URL path prefix where Swagger UI is served (e.g., "/swagger")
	BasePath string
	// SpecPath is the URL path to serve the OpenAPI spec (e.g., "/openapi/specs")
	SpecPath string
	// ResourcesPath is the URL path to serve the resources list (e.g., "/openapi/resources")
	ResourcesPath string
	// Specs is a map of spec name to spec content (YAML or JSON bytes)
	Specs map[string][]byte
	// DefaultSpec is the name of the default spec to serve when no query param is provided
	DefaultSpec string
}

// Handler serves Swagger UI and OpenAPI specifications.
type Handler struct {
	config        Config
	swaggerUI     fs.FS
	resourcesJSON []byte
}

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

	// Build resources list in deterministic order
	names := make([]string, 0, len(config.Specs))
	for name := range config.Specs {
		names = append(names, name)
	}
	slices.Sort(names)

	var resources []Resource
	for _, name := range names {
		resources = append(resources, Resource{
			Name: name,
			URL:  config.SpecPath + "?spec=" + name,
		})
	}

	resourcesJSON, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}

	return &Handler{
		config:        config,
		swaggerUI:     zipReader,
		resourcesJSON: resourcesJSON,
	}, nil
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
			http.Redirect(w, r, r.RequestURI+"/", http.StatusMovedPermanently)
			return
		}
	}

	filePath := strings.TrimPrefix(r.URL.Path, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	h.serveFile(w, filePath)
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
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	h.serveFile(w, filePath)
}

func (h *Handler) serveFile(w http.ResponseWriter, filePath string) {
	file, err := h.swaggerUI.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	// Set content type based on extension
	ext := path.Ext(filePath)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	}

	_, _ = io.Copy(w, file)
}

func (h *Handler) serveSpec(w http.ResponseWriter, r *http.Request) {
	specName := r.URL.Query().Get("spec")
	if specName == "" {
		specName = h.config.DefaultSpec
	}

	spec, ok := h.config.Specs[specName]
	if !ok {
		// If no specific spec requested and we have a default, use it
		if len(h.config.Specs) > 0 && specName == "" {
			for _, s := range h.config.Specs {
				spec = s
				break
			}
		} else {
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(spec)
}

func (h *Handler) serveResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.resourcesJSON)
}
