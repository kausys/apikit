package extractors

import (
	"fmt"

	"github.com/kausys/apikit/cmd/handler/parser"
)

// HTTPExtractor implements FrameworkExtractor for net/http
type HTTPExtractor struct{}

func (e *HTTPExtractor) Name() string {
	return "http"
}

func (e *HTTPExtractor) Imports() []string {
	return []string{"net/http"}
}

func (e *HTTPExtractor) HandlerSignature() string {
	return "func(w http.ResponseWriter, r *http.Request)"
}

func (e *HTTPExtractor) ParseFuncSignature() string {
	return "func parse%s(w http.ResponseWriter, r *http.Request, payload *%s) error"
}

func (e *HTTPExtractor) ExtractQuery(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.URL.Query().Get("%s")`, paramName)
	return GenerateCodeByType(varName, fieldName, GetBaseType(field), field)
}

func (e *HTTPExtractor) ExtractQuerySlice(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.URL.Query()["%s"]`, paramName)
	return GenerateSliceCodeByType(varName, fieldName, field.SliceType, field)
}

func (e *HTTPExtractor) ExtractPath(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.PathValue("%s")`, paramName)
	return GenerateCodeByType(varName, fieldName, GetBaseType(field), field)
}

func (e *HTTPExtractor) ExtractHeader(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.Header.Get("%s")`, paramName)
	return GenerateCodeByType(varName, fieldName, GetBaseType(field), field)
}

func (e *HTTPExtractor) ExtractHeaderSlice(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.Header["%s"]`, paramName)
	return GenerateSliceCodeByType(varName, fieldName, field.SliceType, field)
}

func (e *HTTPExtractor) ExtractCookie(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`apikit.GetCookie(r, "%s")`, paramName)
	return GenerateCodeByType(varName, fieldName, GetBaseType(field), field)
}

func (e *HTTPExtractor) ExtractForm(field *parser.Field, paramName, fieldName string) (string, []string) {
	varName := fmt.Sprintf(`r.FormValue("%s")`, paramName)
	return GenerateCodeByType(varName, fieldName, field.Type, field)
}

func (e *HTTPExtractor) ExtractFormSlice(field *parser.Field, paramName, fieldName string) (string, []string) {
	code := fmt.Sprintf(`if vals := r.Form["%s"]; len(vals) > 0 {
		payload.%s = vals
	}`, paramName, fieldName)
	return code, nil
}

func (e *HTTPExtractor) ExtractFormFile(field *parser.Field, paramName, fieldPath string) (string, []string) {
	code := fmt.Sprintf(`if _, header, err := r.FormFile("%s"); err == nil {
		payload.%s = header
	} else if err != http.ErrMissingFile {
		return fmt.Errorf("reading file '%s': %%w", err)
	}`, paramName, fieldPath, paramName)
	return code, []string{"mime/multipart"}
}

func (e *HTTPExtractor) ExtractFormFiles(field *parser.Field, paramName, fieldPath string) (string, []string) {
	code := fmt.Sprintf(`if form := r.MultipartForm; form != nil {
		if files := form.File["%s"]; len(files) > 0 {
			payload.%s = files
		}
	}`, paramName, fieldPath)
	return code, []string{"mime/multipart"}
}

func (e *HTTPExtractor) ExtractRequest(field *parser.Field) (string, []string) {
	return fmt.Sprintf("payload.%s = r", field.Name), nil
}

func (e *HTTPExtractor) ExtractResponse(field *parser.Field) (string, []string) {
	return fmt.Sprintf("payload.%s = w", field.Name), nil
}

func (e *HTTPExtractor) ParseBody(structName string) string {
	return `if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		return fmt.Errorf("decoding body: %w", err)
	}`
}

func (e *HTTPExtractor) WriteJSON(varName string) string {
	return fmt.Sprintf("apikit.HandleResponse(w, %s, err)", varName)
}

func (e *HTTPExtractor) WriteError(statusCode int, message string) string {
	return fmt.Sprintf(`http.Error(w, "%s", %d)`, message, statusCode)
}
