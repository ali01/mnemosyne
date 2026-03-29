package vault

import (
	"path/filepath"
	"strings"
)

// LinkResolver resolves WikiLink targets to actual file IDs
type LinkResolver struct {
	// Maps for efficient lookups
	pathToID        map[string]string   // Full path -> ID
	basenameToIDs   map[string][]string // Basename -> []IDs (multiple files can have same name)
	normalizedToIDs map[string][]string // Normalized name -> []IDs
	idToPath        map[string]string   // ID -> Full path
}

// NewLinkResolver creates a new link resolver
func NewLinkResolver() *LinkResolver {
	return &LinkResolver{
		pathToID:        make(map[string]string),
		basenameToIDs:   make(map[string][]string),
		normalizedToIDs: make(map[string][]string),
		idToPath:        make(map[string]string),
	}
}

// AddFile registers a file with the resolver
func (r *LinkResolver) AddFile(file *MarkdownFile) {
	id := file.GetID()
	path := file.Path

	// Remove .md extension for matching
	pathWithoutExt := strings.TrimSuffix(path, ".md")

	// Store full path mapping
	r.pathToID[pathWithoutExt] = id
	r.idToPath[id] = path

	// Store basename mapping
	basename := filepath.Base(pathWithoutExt)
	r.basenameToIDs[basename] = append(r.basenameToIDs[basename], id)

	// Store normalized mapping (lowercase, no special chars)
	normalized := normalizeForMatching(basename)
	r.normalizedToIDs[normalized] = append(r.normalizedToIDs[normalized], id)
}

// ResolveLink resolves a WikiLink target to a file ID
func (r *LinkResolver) ResolveLink(target string, sourceFile string) (string, bool) {
	target = strings.TrimSpace(target)

	// Try exact path matches
	if id, found := r.tryExactMatch(target); found {
		return id, true
	}

	// Try relative path match
	targetWithoutExt := strings.TrimSuffix(target, ".md")
	if id, found := r.tryRelativeMatch(targetWithoutExt, sourceFile); found {
		return id, true
	}

	// Try basename and normalized matches
	basename := filepath.Base(targetWithoutExt)
	if id, found := r.tryBasenameMatch(basename, sourceFile); found {
		return id, true
	}

	if id, found := r.tryNormalizedMatch(basename, sourceFile); found {
		return id, true
	}

	return "", false
}

// tryExactMatch attempts exact path matching
func (r *LinkResolver) tryExactMatch(target string) (string, bool) {
	if id, found := r.pathToID[target]; found {
		return id, true
	}

	targetWithoutExt := strings.TrimSuffix(target, ".md")
	if id, found := r.pathToID[targetWithoutExt]; found {
		return id, true
	}

	return "", false
}

// tryRelativeMatch attempts relative path matching
func (r *LinkResolver) tryRelativeMatch(targetWithoutExt, sourceFile string) (string, bool) {
	if sourceFile == "" {
		return "", false
	}

	sourceDir := filepath.Dir(sourceFile)
	relativePath := filepath.Clean(filepath.Join(sourceDir, targetWithoutExt))
	if id, found := r.pathToID[relativePath]; found {
		return id, true
	}

	return "", false
}

// tryBasenameMatch attempts basename matching with preference logic
func (r *LinkResolver) tryBasenameMatch(basename, sourceFile string) (string, bool) {
	ids, found := r.basenameToIDs[basename]
	if !found {
		return "", false
	}

	return r.selectBestMatch(ids, sourceFile)
}

// tryNormalizedMatch attempts normalized/fuzzy matching
func (r *LinkResolver) tryNormalizedMatch(basename, sourceFile string) (string, bool) {
	normalized := normalizeForMatching(basename)
	ids, found := r.normalizedToIDs[normalized]
	if !found {
		return "", false
	}

	return r.selectBestMatch(ids, sourceFile)
}

// selectBestMatch selects the best match from multiple candidates
func (r *LinkResolver) selectBestMatch(ids []string, sourceFile string) (string, bool) {
	if len(ids) == 0 {
		return "", false
	}

	if len(ids) == 1 {
		return ids[0], true
	}

	// Prefer files in the same directory as the source
	if sourceFile != "" {
		sourceDir := filepath.Dir(sourceFile)
		for _, id := range ids {
			if path, ok := r.idToPath[id]; ok {
				if filepath.Dir(path) == sourceDir {
					return id, true
				}
			}
		}
	}

	// Return the first match as fallback
	return ids[0], true
}

// ResolveLinks resolves multiple WikiLinks and returns a map of resolved and unresolved links
func (r *LinkResolver) ResolveLinks(links []WikiLink, sourceFile string) (resolved map[string]string, unresolved []WikiLink) {
	resolved = make(map[string]string)

	for _, link := range links {
		if id, found := r.ResolveLink(link.Target, sourceFile); found {
			resolved[link.Target] = id
		} else {
			unresolved = append(unresolved, link)
		}
	}

	return resolved, unresolved
}

// GetPath returns the file path for a given ID
func (r *LinkResolver) GetPath(id string) (string, bool) {
	path, found := r.idToPath[id]
	return path, found
}

// GetStats returns resolver statistics
func (r *LinkResolver) GetStats() map[string]int {
	return map[string]int{
		"total_files":      len(r.idToPath),
		"unique_basenames": len(r.basenameToIDs),
		"duplicate_names":  r.countDuplicates(),
	}
}

// countDuplicates counts files with duplicate basenames
func (r *LinkResolver) countDuplicates() int {
	count := 0
	for _, ids := range r.basenameToIDs {
		if len(ids) > 1 {
			count += len(ids) - 1
		}
	}
	return count
}

// normalizeForMatching normalizes a string for fuzzy matching
func normalizeForMatching(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Remove common prefixes
	s = strings.TrimPrefix(s, "~")
	s = strings.TrimPrefix(s, "+")

	// Replace separators with spaces
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Remove extra spaces
	s = strings.Join(strings.Fields(s), " ")

	return s
}
