package spec

// OAuth Flow Object
//
// # Configuration details for a supported OAuth Flow
//
// See: https://spec.openapis.org/oas/v3.0.4.html#oauth-flow-object
type OAuthFlow struct {
	// REQUIRED (oauth2 "implicit", "authorizationCode"). The authorization URL to be used for this flow.
	// This MUST be in the form of a URL. The OAuth2 standard requires the use of TLS.
	AuthorizationURL string `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	// REQUIRED (oauth2 "password", "clientCredentials", "authorizationCode"). The token URL to be used
	// for this flow. This MUST be in the form of a URL. The OAuth2 standard requires the use of TLS.
	TokenURL string `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	// The URL to be used for obtaining refresh tokens. This MUST be in the form of a URL. The OAuth2
	// standard requires the use of TLS.
	RefreshURL string `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	// REQUIRED. The available scopes for the OAuth2 security scheme. A map between the scope name and
	// a short description for it. The map MAY be empty.
	Scopes map[string]string `json:"scopes" yaml:"scopes"`
}
