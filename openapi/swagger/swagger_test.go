package swagger

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestZip creates a minimal test zip file for testing.
func createTestZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	files := map[string]string{
		"index.html":             "<html><head></head><body>Swagger UI</body></html>",
		"swagger-ui.css":         ".swagger-ui {}",
		"swagger-ui-bundle.js":   "window.SwaggerUIBundle = {};",
		"swagger-initializer.js": "window.onload = function() {};",
	}

	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestNew(t *testing.T) {
	zipData := createTestZip(t)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "default config",
			config: Config{
				Specs:       map[string][]byte{"api": []byte("openapi: 3.0.0")},
				DefaultSpec: "api",
			},
			wantErr: false,
		},
		{
			name: "custom paths",
			config: Config{
				BasePath:      "/docs",
				SpecPath:      "/api/spec",
				ResourcesPath: "/api/resources",
				Specs:         map[string][]byte{"v1": []byte("openapi: 3.0.0")},
			},
			wantErr: false,
		},
		{
			name: "multiple specs",
			config: Config{
				Specs: map[string][]byte{
					"api":    []byte("openapi: 3.0.0"),
					"admin":  []byte("openapi: 3.0.0"),
					"public": []byte("openapi: 3.0.0"),
				},
				DefaultSpec: "api",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := New(zipData, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, handler)
		})
	}
}

func TestNewInvalidZip(t *testing.T) {
	_, err := New([]byte("invalid zip data"), Config{})
	assert.Error(t, err)
}

func TestHandler_ServeResources(t *testing.T) {
	zipData := createTestZip(t)
	handler, err := New(zipData, Config{
		Specs: map[string][]byte{
			"api":   []byte("openapi: 3.0.0"),
			"admin": []byte("openapi: 3.0.1"),
		},
		DefaultSpec: "api",
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi/resources", nil)
	w := httptest.NewRecorder()

	handler.serveResources(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "api")
	assert.Contains(t, w.Body.String(), "admin")
}

func TestHandler_ServeSpec(t *testing.T) {
	zipData := createTestZip(t)
	handler, err := New(zipData, Config{
		Specs: map[string][]byte{
			"api":   []byte("openapi: 3.0.0 api"),
			"admin": []byte("openapi: 3.0.0 admin"),
		},
		DefaultSpec: "api",
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		query        string
		expectedBody string
		expectedCode int
	}{
		{
			name:         "specific spec",
			query:        "?spec=api",
			expectedBody: "openapi: 3.0.0 api",
			expectedCode: http.StatusOK,
		},
		{
			name:         "admin spec",
			query:        "?spec=admin",
			expectedBody: "openapi: 3.0.0 admin",
			expectedCode: http.StatusOK,
		},
		{
			name:         "default spec",
			query:        "",
			expectedBody: "openapi: 3.0.0 api",
			expectedCode: http.StatusOK,
		},
		{
			name:         "unknown spec",
			query:        "?spec=unknown",
			expectedBody: "",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi/specs"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.serveSpec(w, req)
			if tt.expectedCode == http.StatusOK {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_ServeSwaggerUI(t *testing.T) {
	zipData := createTestZip(t)
	handler, err := New(zipData, Config{
		BasePath: "/swagger",
		Specs:    map[string][]byte{"api": []byte("openapi: 3.0.0")},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "index.html",
			path:           "/swagger/index.html",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
		},
		{
			name:           "root path serves index",
			path:           "/swagger/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
		},
		{
			name:           "css file",
			path:           "/swagger/swagger-ui.css",
			expectedStatus: http.StatusOK,
			expectedType:   "text/css",
		},
		{
			name:           "js file",
			path:           "/swagger/swagger-ui-bundle.js",
			expectedStatus: http.StatusOK,
			expectedType:   "application/javascript",
		},
		{
			name:           "not found",
			path:           "/swagger/nonexistent.js",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.serveSwaggerUI(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	zipData := createTestZip(t)
	handler, err := New(zipData, Config{
		BasePath:      "/swagger",
		SpecPath:      "/openapi/specs",
		ResourcesPath: "/openapi/resources",
		Specs:         map[string][]byte{"api": []byte("openapi: 3.0.0")},
		DefaultSpec:   "api",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "swagger ui",
			path:           "/swagger/index.html",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "specs endpoint",
			path:           "/openapi/specs",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "resources endpoint",
			path:           "/openapi/resources",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unknown path",
			path:           "/unknown",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHandler_Routes(t *testing.T) {
	zipData := createTestZip(t)
	handler, err := New(zipData, Config{
		BasePath:      "/swagger",
		SpecPath:      "/openapi/specs",
		ResourcesPath: "/openapi/resources",
		Specs:         map[string][]byte{"api": []byte("openapi: 3.0.0")},
	})
	require.NoError(t, err)

	mux := http.NewServeMux()
	handler.Routes(mux)

	// Test that routes are registered
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi/specs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi/resources", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
