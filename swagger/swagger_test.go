package swagger

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundle(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := zw.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func newHandler(t *testing.T, cfg Config) *Handler {
	t.Helper()
	h, err := New(bundle(t, map[string]string{
		"index.html":             "<html>ui</html>",
		"swagger-ui.css":         "body{}",
		"swagger-initializer.js": "/* bundled */",
	}), cfg)
	require.NoError(t, err)
	return h
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil))
	return w
}

func TestHandler_ServesTheBundle(t *testing.T) {
	is := assert.New(t)
	h := newHandler(t, Config{})

	index := get(h, "/swagger/")
	css := get(h, "/swagger/swagger-ui.css")

	is.Equal(http.StatusOK, index.Code)
	is.Equal("<html>ui</html>", index.Body.String())
	is.Contains(index.Header().Get("Content-Type"), "text/html")
	is.Contains(css.Header().Get("Content-Type"), "text/css")
	is.Equal(http.StatusMovedPermanently, get(h, "/swagger").Code)
	is.Equal(http.StatusNotFound, get(h, "/swagger/missing.js").Code)
}

// The initializer is written from the Config, so the UI asks for the specs
// where this handler serves them — a SpecPath other than the default included.
func TestHandler_InitializerFollowsTheConfig(t *testing.T) {
	is := assert.New(t)
	h := newHandler(t, Config{
		SpecPath:    "/docs/specs",
		Specs:       []Spec{{Name: "staff & admin", Data: []byte("openapi: 3.1.0")}, {Name: "public", Data: []byte("{}")}},
		DefaultSpec: "public",
	})

	js := get(h, "/swagger/swagger-initializer.js").Body.String()

	is.NotContains(js, "bundled")
	is.Contains(js, `"/docs/specs?spec=staff+%26+admin"`)
	is.Contains(js, `"urls.primaryName": "public"`)
	is.NotContains(js, "location.search", "the page URL must not choose the spec")
}

// A spec name is written into a script: it must not be able to end it.
func TestHandler_InitializerEscapesSpecNames(t *testing.T) {
	h := newHandler(t, Config{Specs: []Spec{{Name: "</script><script>alert(1)//", Data: []byte("x")}}})

	assert.NotContains(t, get(h, "/swagger/swagger-initializer.js").Body.String(), "</script>")
}

func TestHandler_BundledInitializerOnRequest(t *testing.T) {
	h := newHandler(t, Config{BundledInitializer: true})

	assert.Equal(t, "/* bundled */", get(h, "/swagger/swagger-initializer.js").Body.String())
}

// A spec name is a query value: one with spaces or an ampersand used to
// produce a URL that named another spec, or none.
func TestHandler_SpecNamesAreEscapedAndServed(t *testing.T) {
	is := assert.New(t)
	h := newHandler(t, Config{Specs: []Spec{{Name: "a & b", Data: []byte("openapi: 3.1.0")}}})

	var resources []Resource
	require.NoError(t, json.Unmarshal(get(h, "/openapi/resources").Body.Bytes(), &resources))
	require.Len(t, resources, 1)
	spec := get(h, resources[0].URL)

	is.Equal(http.StatusOK, spec.Code)
	is.Equal("openapi: 3.1.0", spec.Body.String())
	is.Equal("application/yaml", spec.Header().Get("Content-Type"))
}

func TestHandler_JSONSpecHasJSONContentType(t *testing.T) {
	h := newHandler(t, Config{Specs: []Spec{{Name: "api", Data: []byte(` {"openapi":"3.1.0"}`)}}})

	assert.Equal(t, "application/json", get(h, "/openapi/specs").Header().Get("Content-Type"))
}

func TestHandler_UnknownSpecIs404(t *testing.T) {
	h := newHandler(t, Config{Specs: []Spec{{Name: "api", Data: []byte("x")}}})

	assert.Equal(t, http.StatusNotFound, get(h, "/openapi/specs?spec=other").Code)
}

func TestNew_RefusesADuplicateSpecName(t *testing.T) {
	_, err := New(bundle(t, map[string]string{"index.html": ""}), Config{
		Specs: []Spec{{Name: "api"}, {Name: "api"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `"api"`)
}

// ServeUI is the mounted form: the router stripped the base path.
func TestHandler_ServeUIUnderAMount(t *testing.T) {
	h := newHandler(t, Config{})
	mux := http.NewServeMux()
	mux.Handle("/docs/", http.StripPrefix("/docs", http.HandlerFunc(h.ServeUI)))

	w := get(mux, "/docs/swagger-ui.css")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.HasPrefix(w.Body.String(), "body"))
}
