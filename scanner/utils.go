package scanner

import (
	"go/ast"
	"regexp"
	"strings"
	"sync"
)

var (
	regexCache sync.Map
)

// getCompiledRegex returns a cached compiled regex for the given pattern.
func getCompiledRegex(pattern string) *regexp.Regexp {
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp)
	}
	re := regexp.MustCompile(pattern)
	regexCache.Store(pattern, re)
	return re
}

// shouldIgnorePath checks if a path should be ignored based on ignore patterns.
func shouldIgnorePath(path string, ignorePaths []string) bool {
	for _, pattern := range ignorePaths {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// hasDirective checks if a comment group contains a specific directive.
func hasDirective(doc *ast.CommentGroup, directive string) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if _, ok := strings.CutPrefix(text, directive); ok {
			return true
		}
	}
	return false
}

// extractDirectiveValue extracts the value after a directive.
// For "swagger:model User", returns "User".
func extractDirectiveValue(doc *ast.CommentGroup, directive string) string {
	if doc == nil {
		return ""
	}
	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if value, ok := strings.CutPrefix(text, directive); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// trimComments extracts all comment lines, removing comment markers.
func trimComments(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	var lines []string
	for _, comment := range doc.List {
		text := comment.Text
		// Handle // comments
		if after, ok := strings.CutPrefix(text, "//"); ok {
			lines = append(lines, strings.TrimSpace(after))
			continue
		}
		// Handle /* */ comments
		if after, ok := strings.CutPrefix(text, "/*"); ok {
			after = strings.TrimSuffix(after, "*/")
			// Split by newlines for multi-line comments
			for line := range strings.SplitSeq(after, "\n") {
				lines = append(lines, strings.TrimSpace(line))
			}
		}
	}
	return lines
}

// extractDescription extracts the description from comments,
// excluding lines that start with known directives.
func extractDescription(doc *ast.CommentGroup, knownDirectives []string) string {
	if doc == nil {
		return ""
	}

	var descLines []string
	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimSpace(text)

		// Skip empty lines and directive lines
		if text == "" {
			continue
		}

		isDirective := false
		for _, dir := range knownDirectives {
			if strings.HasPrefix(text, dir) {
				isDirective = true
				break
			}
		}

		if !isDirective {
			descLines = append(descLines, text)
		}
	}

	return strings.Join(descLines, " ")
}

// extractSectionLines extracts lines from a section until the next directive.
func extractSectionLines(doc *ast.CommentGroup, directive string) []string {
	if doc == nil {
		return nil
	}

	comments := trimComments(doc)
	inSection := false
	var lines []string

	for _, comment := range comments {
		comment = strings.TrimSpace(comment)

		// Check if we're entering the section
		if strings.HasPrefix(comment, directive) {
			inSection = true
			continue
		}

		if !inSection {
			continue
		}

		// Check if we've hit another directive
		if comment != "" && !strings.HasPrefix(comment, DashPrefix) &&
			strings.Contains(comment, ":") && !strings.HasPrefix(comment, "- ") {
			break
		}

		if comment == "" {
			continue
		}

		lines = append(lines, comment)
	}

	return lines
}

// extractSpecs extracts the spec names from the "spec:" directive in comments.
// Format: spec: name1 name2 name3
// Returns nil if no spec directive is found.
func extractSpecs(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		if value, ok := strings.CutPrefix(strings.ToLower(text), SpecDirective); ok {
			value = strings.TrimSpace(value)

			if value == "" {
				return nil
			}

			// Split by spaces
			specs := strings.Fields(value)
			// Normalize: lowercase and trim
			for i, s := range specs {
				specs[i] = strings.TrimSpace(strings.ToLower(s))
			}
			return specs
		}
	}

	return nil
}
