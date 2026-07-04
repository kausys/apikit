package scanner

import "testing"

// A non-dashed `name:` property on an apiKey scheme sets the parameter name
// (header/query/cookie), distinct from the `- name:` line that is the scheme key.
func TestParseSecuritySchemesApiKeyName(t *testing.T) {
	comments := []string{
		"SecuritySchemes:",
		"- name: cookieAuth",
		"type: apiKey",
		"in: cookie",
		"name: Authorization",
		"- name: bearerAuth",
		"type: http",
		"scheme: bearer",
		"bearerFormat: JWT",
	}

	schemes := parseSecuritySchemes(comments, 0)

	ck, ok := schemes["cookieAuth"]
	if !ok {
		t.Fatalf("cookieAuth scheme missing: %v", schemes)
	}
	if ck.Type != "apiKey" || ck.In != "cookie" || ck.Name != "Authorization" {
		t.Errorf("cookieAuth: got type=%q in=%q name=%q, want apiKey/cookie/Authorization",
			ck.Type, ck.In, ck.Name)
	}

	br, ok := schemes["bearerAuth"]
	if !ok {
		t.Fatalf("bearerAuth scheme missing")
	}
	if br.Scheme != "bearer" || br.BearerFormat != "JWT" || br.Name != "" {
		t.Errorf("bearerAuth: got scheme=%q bearerFormat=%q name=%q", br.Scheme, br.BearerFormat, br.Name)
	}
}
