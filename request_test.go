package apikit

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A body over the limit is a 413, not a body cut at the limit. It used to be
// cut, and the truncated JSON was reported as malformed.
func TestReadBody_OverTheLimitIs413(t *testing.T) {
	SetMaxBodyBytes(8)
	t.Cleanup(func() { SetMaxBodyBytes(0) })
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"name":"too long"}`))

	_, err := ReadBody(httptest.NewRecorder(), r)

	require.Error(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, statusOf(err))
	assert.Equal(t, http.StatusRequestEntityTooLarge, statusOf(RequestError(err)), "RequestError keeps the status")
}

func TestReadBody_WithinTheLimit(t *testing.T) {
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{"a":1}`))

	body, err := ReadBody(httptest.NewRecorder(), r)

	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(body))
}

func TestParseMultipartForm_OverTheLimitIs413(t *testing.T) {
	SetMaxMultipartBytes(64)
	t.Cleanup(func() { SetMaxMultipartBytes(0) })
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("note", strings.Repeat("x", 1024)))
	require.NoError(t, mw.Close())
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	err := ParseMultipartForm(httptest.NewRecorder(), r, 32<<20)

	require.Error(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, statusOf(err))
}

func TestRequestError_PlainFailureIs400(t *testing.T) {
	assert.Equal(t, http.StatusBadRequest, statusOf(RequestError(assert.AnError)))
}

// encoding/json/v2: names match exactly, a repeated member is refused.
func TestDecodeJSON_IsStrict(t *testing.T) {
	type body struct {
		Email string `json:"email"`
	}
	is := assert.New(t)

	var exact body
	require.NoError(t, DecodeJSON([]byte(`{"email":"a@b.c","unknown":1}`), &exact))
	is.Equal("a@b.c", exact.Email, "unknown members are ignored")

	var folded body
	require.NoError(t, DecodeJSON([]byte(`{"EMAIL":"a@b.c"}`), &folded))
	is.Empty(folded.Email, "a name in another case is another name")

	var dup body
	require.Error(t, DecodeJSON([]byte(`{"email":"a@b.c","email":"x@y.z"}`), &dup))

	var bad body
	require.Error(t, DecodeJSON([]byte("{\"email\":\"\xff\"}"), &bad))
}

// Responses are written with encoding/json/v2: a nil slice is [] and map keys
// come out sorted.
func TestHandleResponse_WritesJSONv2(t *testing.T) {
	type page struct {
		Items []string        `json:"items"`
		Meta  map[string]int  `json:"meta"`
		Skip  string          `json:"skip,omitempty"`
		Zero  int             `json:"zero,omitzero"`
		Raw   map[string]bool `json:"raw"`
	}
	w := httptest.NewRecorder()

	HandleResponse(w, page{Raw: map[string]bool{"b": true, "a": false}}, nil)

	// The bytes, not an equivalent document: the order of the keys is part
	// of what this proves.
	assert.Equal(t, `{"items":[],"meta":{},"raw":{"a":false,"b":true}}`+"\n", w.Body.String()) //nolint:testifylint // encoded-compare: byte order is the point
}
