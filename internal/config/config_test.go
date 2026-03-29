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
port: 8080
vaults:
  - /path/to/vault1
  - /path/to/vault2
`), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.Len(t, cfg.Vaults, 2)
	assert.Equal(t, "/path/to/vault1", cfg.Vaults[0])
}

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("vaults:\n  - /my/vault\n"), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 5555, cfg.Port)
}

func TestLoadConfigMissingVaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("port: 8080\n"), 0o644)

	_, err := Load(cfgPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vault")
}

func TestLoadConfigExpandsHome(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("vaults:\n  - ~/my-vault\n"), 0o644)

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.NotContains(t, cfg.Vaults[0], "~")
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestDefaultConfigPath(t *testing.T) {
	p := DefaultConfigPath()
	assert.Contains(t, p, "mnemosyne")
	assert.Contains(t, p, "config.yaml")
}

func TestDBPath(t *testing.T) {
	p := DBPath()
	assert.Contains(t, p, "mnemosyne.db")
}

func TestCreateDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "sub", "config.yaml")

	require.NoError(t, CreateDefault(cfgPath, "/my/vault"))

	// File should exist and be loadable
	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, 5555, cfg.Port)
	assert.Equal(t, []string{"/my/vault"}, cfg.Vaults)
}

func TestExpandHome(t *testing.T) {
	expanded := ExpandHome("~/foo")
	assert.NotContains(t, expanded, "~")
	assert.Contains(t, expanded, "foo")

	// Non-tilde path unchanged
	assert.Equal(t, "/abs/path", ExpandHome("/abs/path"))
}
