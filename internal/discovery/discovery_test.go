package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func TestDiscoverRootOnly(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "GRAPH.yaml", "")

	defs, err := Discover(dir)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "", defs[0].RootPath)
	assert.Equal(t, "root", defs[0].Name)
}

func TestDiscoverSiblingGraphs(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "concepts/GRAPH.yaml", "")
	createFile(t, dir, "projects/GRAPH.yaml", "name: My Projects\n")

	defs, err := Discover(dir)
	require.NoError(t, err)
	assert.Len(t, defs, 2)

	names := map[string]string{}
	for _, d := range defs {
		names[d.RootPath] = d.Name
	}
	assert.Equal(t, "concepts", names["concepts"])
	assert.Equal(t, "My Projects", names["projects"])
}

func TestDiscoverNestedError(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "GRAPH.yaml", "")
	createFile(t, dir, "concepts/GRAPH.yaml", "")

	_, err := Discover(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nested")
}

func TestDiscoverDeepNestedError(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "a/GRAPH.yaml", "")
	createFile(t, dir, "a/b/GRAPH.yaml", "")

	_, err := Discover(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nested")
}

func TestDiscoverSkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "GRAPH.yaml", "")
	createFile(t, dir, ".obsidian/GRAPH.yaml", "")

	// Only the root one found, no nesting error from hidden dir
	defs, err := Discover(dir)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
}

func TestDiscoverWithNodeClassification(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "GRAPH.yaml", `
node_classification:
  default_node_type: note
  node_types:
    hub:
      display_name: Hub
      color: "#4ECDC4"
  classification_rules:
    - name: hub_prefix
      priority: 2
      type: filename_prefix
      pattern: "~"
      node_type: hub
`)

	defs, err := Discover(dir)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	require.NotNil(t, defs[0].NodeClassification)
	assert.Equal(t, "note", defs[0].NodeClassification.DefaultNodeType)
	assert.Len(t, defs[0].NodeClassification.ClassificationRules, 1)
}

func TestDiscoverEmptyGraphYAML(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "GRAPH.yaml", "")

	defs, err := Discover(dir)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Nil(t, defs[0].NodeClassification)
}

func TestDiscoverNoGraphYAML(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "notes.md", "hello")

	defs, err := Discover(dir)
	require.NoError(t, err)
	assert.Nil(t, defs)
}

func TestIsUnderPathRoot(t *testing.T) {
	assert.True(t, IsUnderPath("any/file.md", ""))
	assert.True(t, IsUnderPath("file.md", ""))
}

func TestIsUnderPathSubdir(t *testing.T) {
	assert.True(t, IsUnderPath("concepts/AI.md", "concepts"))
	assert.True(t, IsUnderPath("concepts/deep/file.md", "concepts"))
	assert.False(t, IsUnderPath("projects/X.md", "concepts"))
	assert.False(t, IsUnderPath("index.md", "concepts"))
}

func TestIsUnderPathNoPartialMatch(t *testing.T) {
	assert.False(t, IsUnderPath("concepts-extra/file.md", "concepts"))
}

func TestIsUnderPathDeep(t *testing.T) {
	assert.True(t, IsUnderPath("a/b/c/file.md", "a/b"))
	assert.True(t, IsUnderPath("a/b/file.md", "a/b"))
	assert.False(t, IsUnderPath("a/file.md", "a/b"))
}
