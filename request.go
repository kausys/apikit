package apikit

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// DefaultMaxBodyBytes is the largest request body a generated handler reads
// unless SetMaxBodyBytes says otherwise.
const DefaultMaxBodyBytes = 10 << 20

// DefaultMaxMultipartBytes is the largest multipart form a generated handler
// accepts unless SetMaxMultipartBytes says otherwise. Larger than a JSON body,
// because it carries files.
const DefaultMaxMultipartBytes = 64 << 20

var (
	maxBodyBytes      atomic.Int64
	maxMultipartBytes atomic.Int64
)

// SetMaxBodyBytes sets the largest request body a generated handler reads. A
// value of zero or less restores DefaultMaxBodyBytes.
func SetMaxBodyBytes(n int64) {
	maxBodyBytes.Store(max(n, 0))
}

// SetMaxMultipartBytes sets the largest multipart form a generated handler
// accepts, files included. A value of zero or less restores
// DefaultMaxMultipartBytes.
func SetMaxMultipartBytes(n int64) {
	maxMultipartBytes.Store(max(n, 0))
}

func limitOr(v *atomic.Int64, def int64) int64 {
	if n := v.Load(); n > 0 {
		return n
	}
	return def
}

// ReadBody reads a request body, refusing one larger than the limit (see
// SetMaxBodyBytes) with 413.
//
// A body over the limit is refused, not cut short. Cut short, a JSON body
// failed to parse and was reported as malformed, and a raw body — one checked
// against a signature, say — reached the handler as if it were whole.
func ReadBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limitOr(&maxBodyBytes, DefaultMaxBodyBytes)))
	if err != nil {
		return nil, bodyError(err)
	}
	return body, nil
}

// ParseMultipartForm parses a multipart request, refusing one larger than the
// limit (see SetMaxMultipartBytes) with 413. maxMemory is how much of it is
// held in memory; the rest goes to temporary files, which is why the whole
// form needs a limit of its own.
func ParseMultipartForm(w http.ResponseWriter, r *http.Request, maxMemory int64) error {
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, limitOr(&maxMultipartBytes, DefaultMaxMultipartBytes))
	}
	if err := r.ParseMultipartForm(maxMemory); err != nil { //nolint:gosec // G120: the body is bounded by MaxBytesReader above
		return bodyError(err)
	}
	return nil
}

func bodyError(err error) error {
	if tooLarge, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return PayloadTooLarge(fmt.Sprintf("request body exceeds %d bytes", tooLarge.Limit)).WithCause(err)
	}
	return fmt.Errorf("reading body: %w", err)
}

// DecodeJSON decodes a request body with the defaults of encoding/json/v2.
// Member names match field names exactly, so {"Email": …} does not fill a
// field named "email"; an object that names a member twice is refused, where
// v1 kept the last one and two readers of the same body could disagree; and
// invalid UTF-8 is refused rather than replaced. Unknown members are ignored.
func DecodeJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// RequestError turns a failure to read a request into the error the client
// gets: one that already carries a status (a body over the limit, 413) is kept,
// anything else is 400.
func RequestError(err error) error {
	var sc statusCoder
	if errors.As(err, &sc) {
		return err
	}
	return BadRequest("failed to parse request").WithCause(err)
}
