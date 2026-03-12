package scanner

import (
	"go/ast"
	"strings"
)

// processMeta processes swagger:meta directive to extract API metadata.
func (s *Scanner) processMeta(_ string, file *ast.File) error {
	for _, cg := range file.Comments {
		if !hasDirective(cg, MetaDirective) {
			continue
		}

		// Parse meta information from comment block
		meta := parseMetaInfo(cg)
		if meta != nil {
			// Extract spec directive
			meta.Specs = extractSpecs(cg)
			s.Metas = append(s.Metas, meta)
			// Keep backward compatibility: first meta without spec becomes s.Meta
			if s.Meta == nil && len(meta.Specs) == 0 {
				s.Meta = meta
			}
		}
	}
	return nil
}

// parseMetaInfo parses metadata from a comment group.
func parseMetaInfo(doc *ast.CommentGroup) *MetaInfo {
	if doc == nil {
		return nil
	}

	meta := &MetaInfo{
		SecuritySchemes: make(map[string]*SecuritySchemeInfo),
	}

	comments := trimComments(doc)

	for i, comment := range comments {
		comment = strings.TrimSpace(comment)

		switch {
		case strings.HasPrefix(comment, TitleDirective):
			meta.Title = strings.TrimSpace(strings.TrimPrefix(comment, TitleDirective))

		case strings.HasPrefix(comment, VersionDirective):
			meta.Version = strings.TrimSpace(strings.TrimPrefix(comment, VersionDirective))

		case strings.HasPrefix(comment, DescriptionDirective):
			meta.Description = parseMultiLineValue(comments, i, DescriptionDirective)

		case strings.HasPrefix(comment, TermsOfServiceDirective):
			meta.TermsOfService = strings.TrimSpace(strings.TrimPrefix(comment, TermsOfServiceDirective))

		case strings.HasPrefix(comment, HostDirective):
			meta.Host = strings.TrimSpace(strings.TrimPrefix(comment, HostDirective))

		case strings.HasPrefix(comment, BasePathDirective):
			meta.BasePath = strings.TrimSpace(strings.TrimPrefix(comment, BasePathDirective))

		case strings.HasPrefix(comment, ContactDirective):
			meta.Contact = parseContact(comments, i)

		case strings.HasPrefix(comment, LicenseDirective):
			meta.License = parseLicense(comments, i)

		case strings.HasPrefix(comment, ExternalDocsDirective):
			meta.ExternalDocs = parseExternalDocs(comments, i)

		case strings.HasPrefix(comment, TagsDirective):
			meta.Tags = parseTags(comments, i)

		case strings.HasPrefix(comment, SecuritySchemesDirective):
			meta.SecuritySchemes = parseSecuritySchemes(comments, i)

		case strings.HasPrefix(comment, ConsumesDirective):
			meta.Consumes = parseListSection(comments, i)

		case strings.HasPrefix(comment, ProducesDirective):
			meta.Produces = parseListSection(comments, i)
		}
	}

	return meta
}

// parseMultiLineValue parses a multi-line value until the next directive.
func parseMultiLineValue(comments []string, startIdx int, directive string) string {
	var lines []string
	firstLine := strings.TrimSpace(strings.TrimPrefix(comments[startIdx], directive))
	if firstLine != "" {
		lines = append(lines, firstLine)
	}

	for i := startIdx + 1; i < len(comments); i++ {
		line := comments[i]
		// Stop at next directive
		if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			break
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}

	return strings.Join(lines, " ")
}

// parseContact parses contact information.
func parseContact(comments []string, startIdx int) *ContactInfo {
	contact := &ContactInfo{}
	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])
		if !strings.HasPrefix(line, "-") {
			break
		}
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		if after, ok := strings.CutPrefix(line, "name:"); ok {
			contact.Name = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "url:"); ok {
			contact.URL = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "email:"); ok {
			contact.Email = strings.TrimSpace(after)
		}
	}
	return contact
}

// parseLicense parses license information.
func parseLicense(comments []string, startIdx int) *LicenseInfo {
	license := &LicenseInfo{}
	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])
		if !strings.HasPrefix(line, "-") {
			break
		}
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		if after, ok := strings.CutPrefix(line, "name:"); ok {
			license.Name = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "url:"); ok {
			license.URL = strings.TrimSpace(after)
		}
	}
	return license
}

// parseExternalDocs parses external documentation.
func parseExternalDocs(comments []string, startIdx int) *ExternalDocsInfo {
	docs := &ExternalDocsInfo{}
	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])
		if !strings.HasPrefix(line, "-") {
			break
		}
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		if after, ok := strings.CutPrefix(line, "description:"); ok {
			docs.Description = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "url:"); ok {
			docs.URL = strings.TrimSpace(after)
		}
	}
	return docs
}

// parseTags parses tag definitions.
func parseTags(comments []string, startIdx int) []*TagInfo {
	var tags []*TagInfo
	var currentTag *TagInfo

	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])

		// Stop at next top-level directive
		if !strings.HasPrefix(line, "-") && strings.Contains(line, ":") {
			break
		}

		if strings.HasPrefix(line, "- name:") {
			if currentTag != nil {
				tags = append(tags, currentTag)
			}
			currentTag = &TagInfo{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "- name:")),
			}
		} else if currentTag != nil && strings.HasPrefix(line, "description:") {
			currentTag.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}

	if currentTag != nil {
		tags = append(tags, currentTag)
	}

	return tags
}

// parseListSection parses a list section (like Consumes: or Produces:).
// Supports items prefixed with "- " (dash) or just indented values.
func parseListSection(comments []string, startIdx int) []string {
	var items []string

	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])

		// Skip empty lines
		if line == "" {
			continue
		}

		// Stop at next top-level directive
		if isTopLevelDirective(line, startIdx, comments) {
			break
		}

		// Remove leading dash if present
		if after, ok := strings.CutPrefix(line, "-"); ok {
			line = strings.TrimSpace(after)
		}

		if line != "" {
			items = append(items, line)
		}
	}

	return items
}

// parseSecuritySchemes parses security scheme definitions.
// Format:
//
//	SecuritySchemes:
//	  - name: bearer
//	    type: http
//	    scheme: bearer
//	    description: Authorization via Bearer Token
func parseSecuritySchemes(comments []string, startIdx int) map[string]*SecuritySchemeInfo {
	schemes := make(map[string]*SecuritySchemeInfo)
	var currentScheme *SecuritySchemeInfo
	var currentName string

	// Known property prefixes for security schemes
	propertyPrefixes := []string{"type:", "description:", "in:", "scheme:", "bearerFormat:", "name:"}

	for i := startIdx + 1; i < len(comments); i++ {
		line := strings.TrimSpace(comments[i])

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if this is a property line (with or without dash prefix)
		isProperty := false
		cleanLine := line
		if after, ok := strings.CutPrefix(cleanLine, "-"); ok {
			cleanLine = strings.TrimSpace(after)
		}
		for _, prefix := range propertyPrefixes {
			if strings.HasPrefix(cleanLine, prefix) {
				isProperty = true
				break
			}
		}

		// Stop at next top-level directive (not a property and not starting with dash)
		if !isProperty && !strings.HasPrefix(line, "-") && isTopLevelDirective(line, startIdx, comments) {
			break
		}

		// New scheme entry starts with "- name:"
		if strings.HasPrefix(line, "- name:") || strings.HasPrefix(cleanLine, "name:") && strings.HasPrefix(line, "-") {
			if currentScheme != nil && currentName != "" {
				schemes[currentName] = currentScheme
			}
			currentName = strings.TrimSpace(strings.TrimPrefix(cleanLine, "name:"))
			currentScheme = &SecuritySchemeInfo{}
			continue
		}

		// Parse properties for current scheme
		if currentScheme != nil {
			switch {
			case strings.HasPrefix(cleanLine, "type:"):
				currentScheme.Type = strings.TrimSpace(strings.TrimPrefix(cleanLine, "type:"))
			case strings.HasPrefix(cleanLine, "description:"):
				currentScheme.Description = strings.TrimSpace(strings.TrimPrefix(cleanLine, "description:"))
			case strings.HasPrefix(cleanLine, "in:"):
				currentScheme.In = strings.TrimSpace(strings.TrimPrefix(cleanLine, "in:"))
			case strings.HasPrefix(cleanLine, "scheme:"):
				currentScheme.Scheme = strings.TrimSpace(strings.TrimPrefix(cleanLine, "scheme:"))
			case strings.HasPrefix(cleanLine, "bearerFormat:"):
				currentScheme.BearerFormat = strings.TrimSpace(strings.TrimPrefix(cleanLine, "bearerFormat:"))
			}
		}
	}

	if currentScheme != nil && currentName != "" {
		schemes[currentName] = currentScheme
	}

	return schemes
}

// isTopLevelDirective checks if a line is a top-level directive that ends a section.
func isTopLevelDirective(line string, _ int, _ []string) bool {
	// Top-level directives we recognize
	topLevel := []string{
		"SecuritySchemes:", "Tags:", "Contact:", "License:",
		"ExternalDocs:", "Consumes:", "Produces:", "Schemes:",
		"swagger:", "Title:", "Version:", "Host:", "BasePath:",
		"TermsOfService:", "description:",
	}

	for _, directive := range topLevel {
		if strings.HasPrefix(line, directive) {
			return true
		}
	}

	return false
}
