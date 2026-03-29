package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte(`
vault_path: /path/to/vault
port: 8080
db_path: /tmp/test.db
node_classification:
  default_node_type: note
  node_types:
    hub:
      display_name: Hub
      color: "#FF0000"
      size_multiplier: 2.0
  classification_rules:
    - name: hub_tag
      priority: 1
      type: tag
      pattern: hub
      node_type: hub
`), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "/path/to/vault", cfg.VaultPath)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "/tmp/test.db", cfg.DBPath)
	assert.Equal(t, "note", cfg.NodeClassification.DefaultNodeType)
	assert.Len(t, cfg.NodeClassification.NodeTypes, 1)
	assert.Len(t, cfg.NodeClassification.ClassificationRules, 1)
}

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("vault_path: /my/vault\n"), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 5555, cfg.Port)
	assert.Contains(t, cfg.DBPath, "mnemosyne.db")
}

func TestLoadConfigMissingVaultPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("port: 8080\n"), 0o644)

	_, err := Load(cfgPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vault_path")
}

func TestLoadConfigExpandsHome(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("vault_path: ~/my-vault\ndb_path: ~/data/mn.db\n"), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.NotContains(t, cfg.VaultPath, "~")
	assert.NotContains(t, cfg.DBPath, "~")
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	assert.Error(t, err)
}
