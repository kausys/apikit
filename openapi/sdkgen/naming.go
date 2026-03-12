package sdkgen

import (
	"strings"
	"unicode"
)

// toPascalCase converts a string to PascalCase.
// Examples: "balance_service" → "BalanceService", "getBalances" → "GetBalances"
func toPascalCase(s string) string {
	words := splitWords(s)
	var b strings.Builder
	for _, w := range words {
		if w == "" {
			continue
		}
		// Handle common initialisms
		if upper := strings.ToUpper(w); commonInitialisms[upper] {
			b.WriteString(upper)
		} else {
			b.WriteString(strings.ToUpper(w[:1]) + strings.ToLower(w[1:]))
		}
	}
	return b.String()
}

// toCamelCase converts a string to camelCase.
// Examples: "balance_service" → "balanceService", "GetBalances" → "getBalances"
func toCamelCase(s string) string {
	p := toPascalCase(s)
	if p == "" {
		return ""
	}
	// Find the end of leading uppercase run for initialisms
	runes := []rune(p)
	if len(runes) <= 1 {
		return strings.ToLower(p)
	}
	// If first two are uppercase, it's an initialism — lowercase the whole initialism
	if unicode.IsUpper(runes[0]) && unicode.IsUpper(runes[1]) {
		i := 0
		for i < len(runes) && unicode.IsUpper(runes[i]) {
			i++
		}
		// Lowercase all but the last uppercase if followed by lowercase
		if i < len(runes) {
			i--
		}
		return strings.ToLower(string(runes[:i])) + string(runes[i:])
	}
	return strings.ToLower(string(runes[0])) + string(runes[1:])
}

// toSnakeCase converts a string to snake_case.
// Examples: "BalanceService" → "balance_service", "getBalances" → "get_balances"
func toSnakeCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case unicode.IsUpper(r):
			if i > 0 && !unicode.IsUpper(runes[i-1]) {
				b.WriteRune('_')
			} else if i > 0 && i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && !unicode.IsUpper(runes[i+1]) {
				b.WriteRune('_')
			}
			b.WriteRune(unicode.ToLower(r))
		case r == '-' || r == ' ':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// splitWords splits a string into words by separators (underscore, hyphen, space)
// and camelCase boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	runes := []rune(s)

	flush := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}

	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.':
			flush()
		case unicode.IsUpper(r):
			// Start a new word on camelCase boundary
			if i > 0 && !unicode.IsUpper(runes[i-1]) && runes[i-1] != '_' && runes[i-1] != '-' && runes[i-1] != ' ' && runes[i-1] != '.' {
				flush()
			} else if i > 0 && unicode.IsUpper(runes[i-1]) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				// Handle "HTTPClient" → "HTTP", "Client"
				flush()
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return words
}

// commonInitialisms is a set of common initialisms that should be all-caps in Go.
var commonInitialisms = map[string]bool{
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"LHS":   true,
	"OK":    true,
	"QPS":   true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"UUID":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSS":   true,
	"YAML":  true,
	"SDK":   true,
	"HMAC":  true,
	"JWT":   true,
	"RSA":   true,
	"PIX":   true,
	"ACH":   true,
	"BRL":   true,
	"MXN":   true,
	"ARS":   true,
	"COP":   true,
	"USD":   true,
	"BTC":   true,
	"ETH":   true,
	"USDT":  true,
	"KYC":   true,
}
