package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_CloneTestRepository(t *testing.T) {
	// Create test git configuration directly
	gitConfig := &Config{
		RepoURL:      "https://github.com/octocat/Hello-World.git",
		Branch:       "master",
		LocalPath:    "test-repo-clone",
		SyncInterval: 5 * time.Minute,
		AutoSync:     false,
		ShallowClone: true,
		SingleBranch: true,
	}

	// Ensure test directory doesn't exist before test
	_ = os.RemoveAll(gitConfig.LocalPath)

	// Clean up after test
	defer func() {
		_ = os.RemoveAll(gitConfig.LocalPath)
	}()

	// Create manager with test repository
	manager, err := NewManager(gitConfig)
	require.NoError(t, err, "Failed to create git manager")

	// Set update callback (not triggered on initial clone)
	manager.SetUpdateCallback(func(files []string) {
		t.Logf("Changed files: %d", len(files))
	})

	// Initialize (clone) repository
	ctx := context.Background()
	err = manager.Initialize(ctx)
	require.NoError(t, err, "Failed to initialize repository")

	// Verify repository was cloned
	assert.DirExists(t, gitConfig.LocalPath)
	assert.FileExists(t, filepath.Join(gitConfig.LocalPath, "README"))

	// Verify last sync time is set
	lastSync := manager.GetLastSync()
	assert.WithinDuration(t, time.Now(), lastSync, 5*time.Second)

	// Verify the repository info
	assert.Equal(t, "https://github.com/octocat/Hello-World.git", gitConfig.RepoURL)
	assert.Equal(t, "master", gitConfig.Branch)
}

func TestManager_OpenExistingTestRepository(t *testing.T) {
	// Create test git configuration directly
	gitConfig := &Config{
		RepoURL:      "https://github.com/octocat/Hello-World.git",
		Branch:       "master",
		LocalPath:    "data/test-repo-clone-2",
		SyncInterval: 5 * time.Minute,
		AutoSync:     false,
		ShallowClone: true,
		SingleBranch: true,
	}

	// Ensure test directory doesn't exist before test
	_ = os.RemoveAll(gitConfig.LocalPath)
	defer func() {
		_ = os.RemoveAll(gitConfig.LocalPath)
	}()

	// First, clone the repository
	manager1, err := NewManager(gitConfig)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager1.Initialize(ctx)
	require.NoError(t, err, "First initialization should succeed")

	// Now create a new manager and try to open the existing repository
	manager2, err := NewManager(gitConfig)
	require.NoError(t, err)

	err = manager2.Initialize(ctx)
	require.NoError(t, err, "Second initialization should open existing repo")

	// Verify it didn't re-clone
	assert.DirExists(t, gitConfig.LocalPath)
}

func TestManager_InvalidRepository(t *testing.T) {
	// Create config with invalid repository
	gitConfig := &Config{
		RepoURL:      "https://github.com/nonexistent/nonexistent-repo-12345.git",
		Branch:       "main",
		LocalPath:    "data/test-invalid-repo",
		SyncInterval: 5 * time.Minute,
		AutoSync:     false,
		ShallowClone: true,
		SingleBranch: true,
	}

	// Clean up if it exists
	_ = os.RemoveAll(gitConfig.LocalPath)
	defer func() {
		_ = os.RemoveAll(gitConfig.LocalPath)
	}()

	// Create manager
	manager, err := NewManager(gitConfig)
	require.NoError(t, err)

	// Initialize should fail
	ctx := context.Background()
	err = manager.Initialize(ctx)
	assert.Error(t, err, "Should fail to clone non-existent repository")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				RepoURL:   "https://github.com/octocat/Hello-World.git",
				Branch:    "master",
				LocalPath: "test-repo",
			},
			wantErr: false,
		},
		{
			name: "missing repo URL",
			config: &Config{
				Branch:    "main",
				LocalPath: "test-repo",
			},
			wantErr: true,
			errMsg:  "repository URL is required",
		},
		{
			name: "missing local path",
			config: &Config{
				RepoURL: "https://github.com/octocat/Hello-World.git",
				Branch:  "main",
			},
			wantErr: true,
			errMsg:  "local path is required",
		},
		{
			name: "missing branch",
			config: &Config{
				RepoURL:   "https://github.com/octocat/Hello-World.git",
				LocalPath: "test-repo",
			},
			wantErr: true,
			errMsg:  "branch is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_GetLocalPath(t *testing.T) {
	// Create test git configuration directly
	gitConfig := &Config{
		RepoURL:      "https://github.com/octocat/Hello-World.git",
		Branch:       "master",
		LocalPath:    "data/test-repo-clone",
		SyncInterval: 5 * time.Minute,
		AutoSync:     false,
		ShallowClone: true,
		SingleBranch: true,
	}

	// Create manager
	manager, err := NewManager(gitConfig)
	require.NoError(t, err)

	// Get local path (this doesn't require initialization)
	localPath := manager.GetLocalPath()

	assert.Equal(t, "data/test-repo-clone", localPath)
}
