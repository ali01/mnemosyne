package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParser(t *testing.T) {
	parser := NewParser("/test/vault", 0, 0) // Use defaults
	assert.NotNil(t, parser)
	assert.Equal(t, "/test/vault", parser.vaultPath)
	assert.Equal(t, 4, parser.concurrency) // Default concurrency
	assert.Equal(t, 100, parser.batchSize) // Default batch size
}

func TestParser_ParseVault_EmptyVault(t *testing.T) {
	tempDir := t.TempDir()
	parser := NewParser(tempDir, 0, 0)

	result, err := parser.ParseVault()
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.Stats.TotalFiles)
	assert.Equal(t, 0, result.Stats.ParsedFiles)
	assert.Equal(t, 0, result.Stats.FailedFiles)
	assert.Empty(t, result.Files)
	assert.Empty(t, result.ParseErrors)
}

func TestParser_ParseVault_WithFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"note1.md": `---
id: "note1"
tags: ["test"]
---
# Note 1
This links to [[note2]].`,
		"note2.md": `---
id: "note2"
---
# Note 2
This links back to [[note1]].`,
		"subdir/note3.md": `---
id: "note3"
---
# Note 3`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()

	require.NoError(t, err)
	assert.Equal(t, 3, result.Stats.TotalFiles)
	assert.Equal(t, 3, result.Stats.ParsedFiles)
	assert.Equal(t, 0, result.Stats.FailedFiles)
	assert.Len(t, result.Files, 3)

	// Verify files were parsed correctly
	fileMap := make(map[string]*MarkdownFile)
	for _, f := range result.Files {
		fileMap[f.Path] = f
	}

	// Check note1
	note1 := fileMap["note1.md"]
	require.NotNil(t, note1)
	assert.Equal(t, "note1", note1.GetID())
	assert.Len(t, note1.Links, 1)
	assert.Equal(t, "note2", note1.Links[0].Target)

	// Check note2
	note2 := fileMap["note2.md"]
	require.NotNil(t, note2)
	assert.Equal(t, "note2", note2.GetID())
	assert.Len(t, note2.Links, 1)
	assert.Equal(t, "note1", note2.Links[0].Target)

	// Check note3
	note3 := fileMap["subdir/note3.md"]
	require.NotNil(t, note3)
	assert.Equal(t, "note3", note3.GetID())
	assert.Empty(t, note3.Links)
}

func TestParser_ParseVault_WithErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with various issues
	files := map[string]string{
		"valid.md": `---
id: "valid"
---
Content`,
		"missing-id.md": `---
tags: ["test"]
---
No ID here`,
		"invalid-yaml.md": `---
id: "test"
invalid: [unclosed
---
Content`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()

	require.NoError(t, err) // ParseVault itself doesn't fail
	assert.Equal(t, 3, result.Stats.TotalFiles)
	assert.Equal(t, 1, result.Stats.ParsedFiles)
	assert.Equal(t, 2, result.Stats.FailedFiles)
	assert.Len(t, result.Files, 1)
	assert.Len(t, result.ParseErrors, 2)

	// Check that valid file was parsed
	var validFile *MarkdownFile
	for _, f := range result.Files {
		if f.GetID() == "valid" {
			validFile = f
			break
		}
	}
	require.NotNil(t, validFile)
	assert.Equal(t, "valid", validFile.GetID())

	// Check error messages
	errorPaths := make(map[string]bool)
	for _, e := range result.ParseErrors {
		errorPaths[e.FilePath] = true
	}
	assert.True(t, errorPaths["missing-id.md"])
	assert.True(t, errorPaths["invalid-yaml.md"])
}

func TestParser_Progress(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple files
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf(`---
id: "note%d"
---
Content %d`, i, i)
		path := filepath.Join(tempDir, fmt.Sprintf("note%d.md", i))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 0, 0)
	// TODO: Add progress callback support
	// parser.SetWorkerCount(1) // Single worker for predictable progress

	// var progressUpdates []int
	// parser.SetProgressCallback(func(current, total int) {
	// 	progressUpdates = append(progressUpdates, current)
	// })

	result, err := parser.ParseVault()
	require.NoError(t, err)
	assert.Equal(t, 5, result.Stats.ParsedFiles)

	// TODO: Test progress updates when callback support is added
	// assert.NotEmpty(t, progressUpdates)
	// assert.Equal(t, 5, progressUpdates[len(progressUpdates)-1])
}

func TestParser_ConcurrentSafety(t *testing.T) {
	tempDir := t.TempDir()

	// Create many files to test concurrent processing
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		content := fmt.Sprintf(`---
id: "note%d"
tags: ["tag%d"]
---
# Note %d
Link to [[note%d]]`, i, i%10, i, (i+1)%fileCount)

		path := filepath.Join(tempDir, fmt.Sprintf("note%d.md", i))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 0, 0)
	// TODO: Add concurrency control
	// parser.SetWorkerCount(20) // High concurrency

	// var progressCount atomic.Int32
	// parser.SetProgressCallback(func(current, total int) {
	// 	progressCount.Add(1)
	// })

	result, err := parser.ParseVault()
	require.NoError(t, err)
	assert.Equal(t, fileCount, result.Stats.TotalFiles)
	assert.Equal(t, fileCount, result.Stats.ParsedFiles)
	assert.Equal(t, 0, result.Stats.FailedFiles)
	assert.Len(t, result.Files, fileCount)

	// Verify all files have unique IDs
	idMap := make(map[string]bool)
	for _, f := range result.Files {
		id := f.GetID()
		assert.False(t, idMap[id], "Duplicate ID found: %s", id)
		idMap[id] = true
	}
}

func TestParser_SkipNonMarkdown(t *testing.T) {
	tempDir := t.TempDir()

	// Create various file types
	files := map[string]string{
		"note.md": `---
id: "note"
---
Content`,
		"image.png":     "binary data",
		"config.yaml":   "key: value",
		".hidden.md":    "hidden file",
		"README.txt":    "text file",
		"subfolder.md/": "", // Directory
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if strings.HasSuffix(path, "/") {
			require.NoError(t, os.MkdirAll(fullPath, 0o750))
		} else {
			require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
		}
	}

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()

	require.NoError(t, err)
	assert.Equal(t, 1, result.Stats.TotalFiles) // Only note.md
	assert.Equal(t, 1, result.Stats.ParsedFiles)
	assert.Len(t, result.Files, 1)
	// Verify the parsed file
	var noteFile *MarkdownFile
	for _, f := range result.Files {
		noteFile = f
		break
	}
	require.NotNil(t, noteFile)
	assert.Equal(t, "note", noteFile.GetID())
}

func TestParseResult_GetFile(t *testing.T) {
	// Test GetFile and GetFileByPath methods
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"note1.md": `---
id: "id1"
---
Content 1`,
		"note2.md": `---
id: "id2"
---
Content 2`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()
	require.NoError(t, err)

	// Test GetFile (by ID)
	file1, found := result.GetFile("id1")
	assert.True(t, found)
	assert.NotNil(t, file1)
	assert.Equal(t, "id1", file1.GetID())
	assert.Equal(t, "note1.md", file1.Path)

	// Test non-existent ID
	fileNil, found := result.GetFile("non-existent")
	assert.False(t, found)
	assert.Nil(t, fileNil)

	// Test GetFileByPath
	file2, found := result.GetFileByPath("note2.md")
	assert.True(t, found)
	assert.NotNil(t, file2)
	assert.Equal(t, "id2", file2.GetID())
	assert.Equal(t, "note2.md", file2.Path)

	// Test non-existent path
	fileNil2, found := result.GetFileByPath("non-existent.md")
	assert.False(t, found)
	assert.Nil(t, fileNil2)
}

func TestParser_CollectMarkdownFiles_Error(t *testing.T) {
	// Test error handling in collectMarkdownFiles
	nonExistentPath := "/non/existent/path"
	parser := NewParser(nonExistentPath, 0, 0)

	result, err := parser.ParseVault()
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to collect markdown files")
}

func TestParser_PermissionError(t *testing.T) {
	// Test handling of permission errors
	if os.Getuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	tempDir := t.TempDir()
	restrictedDir := filepath.Join(tempDir, "restricted")
	require.NoError(t, os.Mkdir(restrictedDir, 0o750))

	// Create a file in the restricted directory
	testFile := filepath.Join(restrictedDir, "test.md")
	content := `---
id: "test"
---
Content`
	require.NoError(t, os.WriteFile(testFile, []byte(content), 0o600))

	// Remove read permissions from directory
	require.NoError(t, os.Chmod(restrictedDir, 0o000))
	defer func() { _ = os.Chmod(restrictedDir, 0o755) }() // #nosec G302 -- Test cleanup requires full permissions

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()

	// Permission error during file collection causes parse to fail
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParser_LargeFile(t *testing.T) {
	// Test parsing a large markdown file
	tempDir := t.TempDir()

	// Create a large file (1MB)
	var contentBuilder strings.Builder
	contentBuilder.WriteString(`---
id: "large-file"
tags: ["performance", "test"]
---
# Large File Test

`)

	// Add 100KB of content
	for i := 0; i < 100; i++ {
		contentBuilder.WriteString(strings.Repeat("This is a test line with some content. ", 25))
		contentBuilder.WriteString("\n")
		if i%10 == 0 {
			contentBuilder.WriteString(fmt.Sprintf("Link to [[note%d]] here.\n", i))
		}
	}

	largePath := filepath.Join(tempDir, "large.md")
	require.NoError(t, os.WriteFile(largePath, []byte(contentBuilder.String()), 0o600))

	parser := NewParser(tempDir, 0, 0)
	result, err := parser.ParseVault()

	require.NoError(t, err)
	assert.Equal(t, 1, result.Stats.ParsedFiles)

	file := result.Files["large-file"]
	require.NotNil(t, file)
	assert.Equal(t, "large-file", file.GetID())
	assert.Len(t, file.Links, 10) // Should have 10 links
}

func TestParser_ConcurrentModification(t *testing.T) {
	// Test that parser handles concurrent file modifications safely
	tempDir := t.TempDir()

	// Create initial files
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`---
id: "note%d"
---
Content %d`, i, i)
		path := filepath.Join(tempDir, fmt.Sprintf("note%d.md", i))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	parser := NewParser(tempDir, 4, 2) // Use multiple workers and small batch size

	// Start parsing in a goroutine
	var result *ParseResult
	var parseErr error
	done := make(chan bool)

	go func() {
		result, parseErr = parser.ParseVault()
		done <- true
	}()

	// Modify files during parsing
	time.Sleep(10 * time.Millisecond)
	for i := 0; i < 5; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("note%d.md", i))
		newContent := fmt.Sprintf(`---
id: "note%d"
---
Modified content %d`, i, i)
		_ = os.WriteFile(path, []byte(newContent), 0o600)
	}

	<-done

	// Should complete without errors
	require.NoError(t, parseErr)
	assert.GreaterOrEqual(t, result.Stats.ParsedFiles, 5)
}
