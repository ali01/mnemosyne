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

	// Extract title from frontmatter or filename
	title := extractTitle(relativePath, frontmatter)

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

	// Extract title from frontmatter or path
	title := extractTitle(path, frontmatter)

	return &MarkdownFile{
		Path:        path,
		Title:       title,
		Content:     contentStr,
		Frontmatter: frontmatter,
		Links:       links,
		FileInfo:    nil, // No file info when processing from reader
	}, nil
}

// extractTitle extracts a title from a file, preferring frontmatter title over filename
func extractTitle(path string, frontmatter *FrontmatterData) string {
	// Check frontmatter first
	if frontmatter != nil {
		if title, ok := frontmatter.GetString("title"); ok && title != "" {
			return title
		}
	}

	// Fall back to filename without .md extension
	if path == "" {
		return ""
	}

	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
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

// GetNodeType determines the node type using the provided NodeClassifier
// If classifier is nil, returns "note" as the default type
func (m *MarkdownFile) GetNodeType(classifier *NodeClassifier) string {
	if classifier == nil {
		return "note"
	}
	return classifier.ClassifyNode(m)
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
