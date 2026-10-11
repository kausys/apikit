// Package generator provides OpenAPI specification generation from Go source code.
package generator

// Config holds configuration options for the generator.
type Config struct {
	// Dir is the root directory of the project
	Dir string
	// Pattern is the package pattern to scan (e.g., "./...", "./api/...")
	Pattern string
	// IgnorePaths contains path patterns to exclude during scanning
	IgnorePaths []string
	// OutputFile is the output file path for the generated spec
	OutputFile string
	// OutputFormat is the output format: "yaml" or "json"
	OutputFormat string
	// Check writes nothing: generation fails when a file on disk is not what
	// the source generates.
	Check bool
	// Validate makes a problem the validation of the generated documents
	// finds fatal; without it the problems are warnings. Strict implies it.
	Validate bool
	// CleanUnused removes schemas that are declared but not referenced
	CleanUnused bool
	// NoDefault skips generating the default spec for routes without spec: directives
	NoDefault bool

	// Strict turns conditions that silently drop routes from the output into
	// errors. Off by default so existing callers keep their behaviour.
	Strict bool
	// EnumRefs generates enums as $ref references to components/schemas instead of inline
	EnumRefs bool
}

// Option is a function type for configuring the Generator.
type Option func(*Config)

// WithDir sets the root directory.
func WithDir(dir string) Option {
	return func(c *Config) {
		c.Dir = dir
	}
}

// WithPattern sets the package pattern to scan.
func WithPattern(pattern string) Option {
	return func(c *Config) {
		c.Pattern = pattern
	}
}

// WithIgnorePaths sets the path patterns to ignore.
func WithIgnorePaths(paths ...string) Option {
	return func(c *Config) {
		c.IgnorePaths = append(c.IgnorePaths, paths...)
	}
}

// WithOutput sets the output file and format.
func WithOutput(file, format string) Option {
	return func(c *Config) {
		c.OutputFile = file
		c.OutputFormat = format
	}
}

// WithCache does nothing. The cache it enabled was written on every run and
// never read: a full scan is what produced every spec either way.
//
// Deprecated: there is no cache.
func WithCache(bool) Option {
	return func(*Config) {}
}

// WithFlatten does nothing; it never did.
//
// Deprecated: schemas are always referenced, not inlined.
func WithFlatten(bool) Option {
	return func(*Config) {}
}

// WithCheck makes generation write nothing and fail when a file on disk is
// not what the source generates — for CI.
func WithCheck(check bool) Option {
	return func(c *Config) {
		c.Check = check
	}
}

// WithValidation enables or disables spec validation.
func WithValidation(validate bool) Option {
	return func(c *Config) {
		c.Validate = validate
	}
}

// WithCleanUnused enables or disables removal of unreferenced schemas.
func WithCleanUnused(clean bool) Option {
	return func(c *Config) {
		c.CleanUnused = clean
	}
}

// WithStrict makes silent route loss fatal (see Config.Strict).
func WithStrict(strict bool) Option {
	return func(c *Config) {
		c.Strict = strict
	}
}

// WithNoDefault skips generating the default spec for routes without spec: directives.
func WithNoDefault(noDefault bool) Option {
	return func(c *Config) {
		c.NoDefault = noDefault
	}
}

// WithEnumRefs enables generating enums as $ref references instead of inline.
func WithEnumRefs(enumRefs bool) Option {
	return func(c *Config) {
		c.EnumRefs = enumRefs
	}
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Dir:         ".",
		Pattern:     "./...",
		IgnorePaths: []string{},
		// Nothing is written unless asked for: a relative default landed in
		// whatever directory the caller ran from — the package directory, for
		// a test.
		OutputFile:   "",
		OutputFormat: "yaml",
		Validate:     false,
		CleanUnused:  true,
	}
}
