package tags

import (
	"go/ast"
	"regexp"
	"strings"

	"github.com/kausys/apikit/openapi/parser"
)

// SingleLineParser is a helper for parsers that extract single-line values.
// It handles patterns like "Summary: This is the summary"
type SingleLineParser struct {
	parser.BaseParser
	pattern *regexp.Regexp
	prefix  string
}

// NewSingleLineParser creates a new single-line parser.
func NewSingleLineParser(name, prefix string, contexts []parser.Context, setters parser.SetterMap) *SingleLineParser {
	// Case-insensitive pattern matching
	pattern := regexp.MustCompile(`(?i)^\s*` + regexp.QuoteMeta(prefix) + `\s*(.*)$`)

	return &SingleLineParser{
		BaseParser: parser.NewBaseParser(name, parser.ParserTypeSingleLine, contexts, setters),
		pattern:    pattern,
		prefix:     strings.ToLower(prefix),
	}
}

// Matches checks if any line starts with the prefix.
func (p *SingleLineParser) Matches(comment string, ctx parser.Context) bool {
	if !p.SupportsContext(ctx) {
		return false
	}

	lower := strings.ToLower(comment)
	return strings.Contains(lower, p.prefix)
}

// Parse extracts the value after the prefix.
func (p *SingleLineParser) Parse(comments *ast.CommentGroup, ctx parser.Context) (any, error) {
	for _, c := range comments.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		if matches := p.pattern.FindStringSubmatch(text); len(matches) > 1 {
			return strings.TrimSpace(matches[1]), nil
		}
	}
	return "", nil
}

// Apply applies the value using the context's setter.
func (p *SingleLineParser) Apply(target, value any, ctx parser.Context) error {
	if value == "" {
		return nil
	}
	return p.ApplyWithSetter(target, value, ctx)
}

// MultiLineParser is a helper for parsers that extract multi-line values.
// It collects all lines after the directive until the next directive.
type MultiLineParser struct {
	parser.BaseParser
	prefix string
}

// NewMultiLineParser creates a new multi-line parser.
func NewMultiLineParser(name, prefix string, contexts []parser.Context, setters parser.SetterMap) *MultiLineParser {
	return &MultiLineParser{
		BaseParser: parser.NewBaseParser(name, parser.ParserTypeMultiLine, contexts, setters),
		prefix:     strings.ToLower(prefix),
	}
}

// Matches checks if the comment contains the prefix.
func (p *MultiLineParser) Matches(comment string, ctx parser.Context) bool {
	if !p.SupportsContext(ctx) {
		return false
	}
	return strings.Contains(strings.ToLower(comment), p.prefix)
}

// Parse extracts multi-line content.
func (p *MultiLineParser) Parse(comments *ast.CommentGroup, ctx parser.Context) (any, error) {
	var lines []string
	collecting := false

	for _, c := range comments.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		lower := strings.ToLower(text)

		if strings.HasPrefix(lower, p.prefix) {
			collecting = true
			// Get content after prefix on same line
			after := strings.TrimSpace(text[len(p.prefix):])
			if after != "" {
				lines = append(lines, after)
			}
			continue
		}

		if collecting {
			// Stop at next directive
			if strings.Contains(text, ":") && !strings.HasPrefix(text, " ") {
				break
			}
			lines = append(lines, text)
		}
	}

	return strings.Join(lines, "\n"), nil
}

// Apply applies the value using the context's setter.
func (p *MultiLineParser) Apply(target, value any, ctx parser.Context) error {
	if value == "" {
		return nil
	}
	return p.ApplyWithSetter(target, value, ctx)
}

// ListParser is a helper for parsers that extract comma or space-separated lists.
type ListParser struct {
	parser.BaseParser
	pattern   *regexp.Regexp
	prefix    string
	separator string
}

// NewListParser creates a new list parser.
func NewListParser(name, prefix, separator string, contexts []parser.Context, setters parser.SetterMap) *ListParser {
	pattern := regexp.MustCompile(`(?i)^\s*` + regexp.QuoteMeta(prefix) + `\s*(.*)$`)

	return &ListParser{
		BaseParser: parser.NewBaseParser(name, parser.ParserTypeList, contexts, setters),
		pattern:    pattern,
		prefix:     strings.ToLower(prefix),
		separator:  separator,
	}
}

// Matches checks if the comment contains the prefix.
func (p *ListParser) Matches(comment string, ctx parser.Context) bool {
	if !p.SupportsContext(ctx) {
		return false
	}
	return strings.Contains(strings.ToLower(comment), p.prefix)
}

// Parse extracts and splits the list.
func (p *ListParser) Parse(comments *ast.CommentGroup, ctx parser.Context) (any, error) {
	for _, c := range comments.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)

		if matches := p.pattern.FindStringSubmatch(text); len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			var items []string
			for item := range strings.SplitSeq(value, p.separator) {
				item = strings.TrimSpace(item)
				if item != "" {
					items = append(items, item)
				}
			}
			return items, nil
		}
	}
	return []string{}, nil
}

// Apply applies the value using the context's setter.
func (p *ListParser) Apply(target, value any, ctx parser.Context) error {
	items, ok := value.([]string)
	if !ok || len(items) == 0 {
		return nil
	}
	return p.ApplyWithSetter(target, value, ctx)
}
