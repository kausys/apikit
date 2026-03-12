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
	// UseCache enables incremental build caching
	UseCache bool
	// Flatten inlines $ref schemas instead of using references
	Flatten bool
	// Validate enables spec validation after generation
	Validate bool
	// CleanUnused removes schemas that are declared but not referenced
	CleanUnused bool
	// NoDefault skips generating the default spec for routes without spec: directives
	NoDefault bool
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

// WithCache enables or disables caching.
func WithCache(enabled bool) Option {
	return func(c *Config) {
		c.UseCache = enabled
	}
}

// WithFlatten enables or disables schema flattening.
func WithFlatten(flatten bool) Option {
	return func(c *Config) {
		c.Flatten = flatten
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
		Dir:          ".",
		Pattern:      "./...",
		IgnorePaths:  []string{},
		OutputFile:   "openapi.yaml",
		OutputFormat: "yaml",
		UseCache:     true,
		Flatten:      false,
		Validate:     true,
		CleanUnused:  true,
	}
}
