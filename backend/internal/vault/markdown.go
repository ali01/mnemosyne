package vault

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownFile represents a parsed markdown file
type MarkdownFile struct {
	Path        string           // Relative path in vault
	Title       string           // Extracted from filename
	Content     string           // Raw markdown content
	Frontmatter *FrontmatterData // Parsed frontmatter
	Links       []WikiLink       // Extracted WikiLinks
	FileInfo    os.FileInfo      // File metadata
}

// ProcessMarkdownFile reads and processes a markdown file
func ProcessMarkdownFile(vaultPath, relativePath string) (*MarkdownFile, error) {
	fullPath := filepath.Join(vaultPath, relativePath)

	// Read file content
	content, err := os.ReadFile(fullPath) // #nosec G304 -- fullPath is from controlled vault directory
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", relativePath, err)
	}

	// Get file info
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", relativePath, err)
	}

	// Process content
	contentStr := string(content)

	// Extract frontmatter
	frontmatter, bodyContent, err := ExtractFrontmatter(contentStr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter from %s: %w", relativePath, err)
	}

	// Extract WikiLinks from body content (not frontmatter)
	links := ExtractWikiLinks(bodyContent)

	// Extract title from filename
	title := extractTitle(relativePath)

	return &MarkdownFile{
		Path:        relativePath,
		Title:       title,
		Content:     contentStr,
		Frontmatter: frontmatter,
		Links:       links,
		FileInfo:    fileInfo,
	}, nil
}

// ProcessMarkdownReader processes markdown from a reader (for testing)
func ProcessMarkdownReader(reader io.Reader, path string) (*MarkdownFile, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	contentStr := string(content)

	// Extract frontmatter
	frontmatter, bodyContent, err := ExtractFrontmatter(contentStr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Extract WikiLinks
	links := ExtractWikiLinks(bodyContent)

	// Extract title from path
	title := extractTitle(path)

	return &MarkdownFile{
		Path:        path,
		Title:       title,
		Content:     contentStr,
		Frontmatter: frontmatter,
		Links:       links,
		FileInfo:    nil, // No file info when processing from reader
	}, nil
}

// extractTitle extracts a clean title from a file path
func extractTitle(path string) string {
	// Handle empty path
	if path == "" {
		return ""
	}

	// Get the base filename
	base := filepath.Base(path)

	// Handle the case where base is "." (from empty path)
	if base == "." {
		return "."
	}

	// Remove .md extension
	title := strings.TrimSuffix(base, ".md")

	// Remove special prefixes (like ~ for hub nodes)
	title = strings.TrimPrefix(title, "~")

	// Replace underscores and hyphens with spaces for better readability
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Capitalize first letter of each word while preserving multiple spaces
	result := ""
	inWord := false
	for _, ch := range title {
		if ch == ' ' {
			result += " "
			inWord = false
		} else if !inWord {
			// First character of a word
			result += strings.ToUpper(string(ch))
			inWord = true
		} else {
			// Rest of the word
			result += strings.ToLower(string(ch))
		}
	}

	return result
}

// GetID returns the unique ID from frontmatter
func (m *MarkdownFile) GetID() string {
	if m.Frontmatter != nil {
		return m.Frontmatter.ID
	}
	return ""
}

// GetTags returns all tags from frontmatter
func (m *MarkdownFile) GetTags() []string {
	if m.Frontmatter != nil {
		return m.Frontmatter.Tags
	}
	return []string{}
}

// GetNodeType determines the node type based on tags and file path
func (m *MarkdownFile) GetNodeType() string {
	// Check frontmatter tags first
	if m.Frontmatter != nil {
		if m.Frontmatter.HasTag("index") {
			return "index"
		}
		if m.Frontmatter.HasTag("open-question") {
			return "question"
		}
	}

	// Check filename prefix
	base := filepath.Base(m.Path)
	if strings.HasPrefix(base, "~") {
		return "hub"
	}

	// Check directory structure
	parts := strings.Split(m.Path, string(filepath.Separator))
	if len(parts) > 0 {
		switch parts[0] {
		case "concepts":
			return "concept"
		case "references":
			return "reference"
		case "projects":
			return "project"
		case "prototypes":
			return "prototype"
		}
	}

	return "default"
}

// GetCreatedAt returns the file creation time
func (m *MarkdownFile) GetCreatedAt() time.Time {
	if m.FileInfo != nil {
		return m.FileInfo.ModTime() // Note: Unix doesn't track creation time, using mod time
	}
	return time.Now()
}

// GetModifiedAt returns the file modification time
func (m *MarkdownFile) GetModifiedAt() time.Time {
	if m.FileInfo != nil {
		return m.FileInfo.ModTime()
	}
	return time.Now()
}
