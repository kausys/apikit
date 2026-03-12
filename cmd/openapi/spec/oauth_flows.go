package spec

// OAuth Flows Object
//
// Allows configuration of the supported OAuth Flows.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#oauth-flows-object
type OAuthFlows struct {
	// Configuration for the OAuth Implicit flow
	Implicit *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	// Configuration for the OAuth Resource Owner Password flow
	Password *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
	// Configuration for the OAuth Client Credentials flow. Previously called application in OpenAPI 2.0.
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	// Configuration for the OAuth Authorization Code flow. Previously called accessCode in OpenAPI 2.0.
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}
