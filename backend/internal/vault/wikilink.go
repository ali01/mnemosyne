package vault

import (
	"regexp"
	"strings"
)

// WikiLink represents a parsed WikiLink from markdown
type WikiLink struct {
	Raw         string // Complete link text with brackets
	Target      string // Target note name
	DisplayText string // Alias text if present
	Section     string // Heading/section if present
	LinkType    string // "wikilink" or "embed"
	Position    int    // Character position
}

var (
	// Matches [[Target]], [[Target|Alias]], ![[Embed]], etc.
	// Updated to handle nested brackets within the link
	wikiLinkRegex = regexp.MustCompile(`(!?)\[\[(.+?)\]\]`)

	// Splits link content: target#section|alias
	// Updated to handle optional target (for section-only links)
	linkPartsRegex = regexp.MustCompile(`^([^#\|]*)(#[^|]+)?(\|(.+))?$`)
)

// ExtractWikiLinks finds all WikiLinks in content
func ExtractWikiLinks(content string) []WikiLink {
	matches := wikiLinkRegex.FindAllStringSubmatchIndex(content, -1)
	links := make([]WikiLink, 0, len(matches))
	for _, match := range matches {
		// match[0], match[1] = full match start/end
		// match[2], match[3] = embed prefix (!) start/end
		// match[4], match[5] = inner content start/end

		if len(match) < 6 {
			continue
		}

		raw := content[match[0]:match[1]]
		isEmbed := match[2] != match[3] // Check if ! prefix exists
		innerContent := content[match[4]:match[5]]

		link := parseWikiLink(raw, innerContent, isEmbed, match[0])
		links = append(links, link)
	}

	return links
}

// parseWikiLink parses the inner content of a WikiLink
func parseWikiLink(raw, innerContent string, isEmbed bool, position int) WikiLink {
	link := WikiLink{
		Raw:      raw,
		Position: position,
	}

	if isEmbed {
		link.LinkType = "embed"
	} else {
		link.LinkType = "wikilink"
	}

	// Handle edge cases first
	if innerContent == "" {
		link.Target = ""
		link.DisplayText = ""
		return link
	}

	if innerContent == "|" {
		link.Target = ""
		link.DisplayText = ""
		return link
	}

	// Parse the inner content
	innerContent = strings.TrimSpace(innerContent)

	// Special handling for section-only links like "#Section"
	if strings.HasPrefix(innerContent, "#") {
		link.Target = ""
		link.Section = strings.TrimPrefix(innerContent, "#")
		link.DisplayText = innerContent
		return link
	}

	parts := linkPartsRegex.FindStringSubmatch(innerContent)

	if len(parts) > 1 {
		// parts[1] = target
		link.Target = strings.TrimSpace(parts[1])

		// parts[2] = #section (including #)
		if len(parts) > 2 && parts[2] != "" {
			link.Section = strings.TrimPrefix(parts[2], "#")
		}

		// parts[4] = alias (after |)
		if len(parts) > 4 && parts[4] != "" {
			link.DisplayText = strings.TrimSpace(parts[4])
		}
	} else {
		// Fallback for simple links
		link.Target = innerContent
	}

	// If no display text, use the target
	if link.DisplayText == "" {
		if link.Section != "" {
			link.DisplayText = link.Target + "#" + link.Section
		} else {
			link.DisplayText = link.Target
		}
	}

	return link
}

// GetUniqueTargets returns a deduplicated list of all link targets
func GetUniqueTargets(links []WikiLink) []string {
	targetMap := make(map[string]bool)
	for _, link := range links {
		targetMap[link.Target] = true
	}

	targets := make([]string, 0, len(targetMap))
	for target := range targetMap {
		targets = append(targets, target)
	}

	return targets
}

// FilterByType returns only links of the specified type
func FilterByType(links []WikiLink, linkType string) []WikiLink {
	filtered := make([]WikiLink, 0, len(links))
	for _, link := range links {
		if link.LinkType == linkType {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

// NormalizeTarget normalizes a link target for matching
// This includes trimming whitespace and converting to lowercase
func NormalizeTarget(target string) string {
	return strings.ToLower(strings.TrimSpace(target))
}
