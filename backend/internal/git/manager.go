package git

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// Manager handles Git repository operations
type Manager struct {
	config     *Config
	repo       *git.Repository
	mu         sync.RWMutex
	syncMu     sync.Mutex // Prevents concurrent syncs
	lastSync   time.Time
	syncTicker *time.Ticker
	stopChan   chan struct{}
	
	// Callbacks
	onUpdate func(changedFiles []string) // Called when files change
}

// NewManager creates a new Git manager
func NewManager(config *Config) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	m := &Manager{
		config:   config,
		stopChan: make(chan struct{}),
	}
	
	return m, nil
}

// Initialize sets up the repository (clone or open existing)
func (m *Manager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if repo already exists
	if _, err := os.Stat(m.config.LocalPath); err == nil {
		// Open existing repository
		repo, err := git.PlainOpen(m.config.LocalPath)
		if err != nil {
			log.Printf("Failed to open existing repo, will re-clone: %v", err)
			// Remove corrupted repo
			os.RemoveAll(m.config.LocalPath)
		} else {
			m.repo = repo
			log.Printf("Opened existing repository at %s", m.config.LocalPath)
			
			// Pull latest changes
			if err := m.pullInternal(ctx); err != nil {
				log.Printf("Warning: Failed to pull latest changes: %v", err)
			}
			return nil
		}
	}
	
	// Clone repository
	log.Printf("Cloning repository from %s to %s", m.config.RepoURL, m.config.LocalPath)
	
	cloneOptions := &git.CloneOptions{
		URL:           m.config.RepoURL,
		Auth:          m.getAuth(),
		Progress:      os.Stdout,
		SingleBranch:  m.config.SingleBranch,
		ReferenceName: plumbing.NewBranchReferenceName(m.config.Branch),
	}
	
	if m.config.ShallowClone {
		cloneOptions.Depth = 1
	}
	
	repo, err := git.PlainCloneContext(ctx, m.config.LocalPath, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}
	
	m.repo = repo
	m.lastSync = time.Now()
	log.Printf("Successfully cloned repository")
	
	return nil
}

// Pull fetches and merges latest changes
func (m *Manager) Pull(ctx context.Context) error {
	// Prevent concurrent pulls
	if !m.syncMu.TryLock() {
		return ErrSyncInProgress
	}
	defer m.syncMu.Unlock()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.pullInternal(ctx)
}

// pullInternal performs the actual pull operation
func (m *Manager) pullInternal(ctx context.Context) error {
	if m.repo == nil {
		return ErrRepoNotFound
	}
	
	worktree, err := m.repo.Worktree()
	if err != nil {
		return err
	}
	
	// Get list of files before pull
	oldFiles := m.getFileList()
	
	// Pull with force to handle conflicts (read-only vault)
	err = worktree.PullContext(ctx, &git.PullOptions{
		RemoteName:    "origin",
		Auth:          m.getAuth(),
		Progress:      os.Stdout,
		Force:         true, // Force pull for read-only vault
		SingleBranch:  m.config.SingleBranch,
		ReferenceName: plumbing.NewBranchReferenceName(m.config.Branch),
	})
	
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	
	m.lastSync = time.Now()
	
	// Get list of files after pull
	newFiles := m.getFileList()
	
	// Find changed files
	changedFiles := m.findChangedFiles(oldFiles, newFiles)
	
	if len(changedFiles) > 0 && m.onUpdate != nil {
		log.Printf("Detected %d changed files", len(changedFiles))
		go m.onUpdate(changedFiles)
	}
	
	if err == git.NoErrAlreadyUpToDate {
		log.Printf("Repository is already up to date")
	} else {
		log.Printf("Successfully pulled latest changes")
	}
	
	return nil
}

// StartAutoSync begins automatic synchronization
func (m *Manager) StartAutoSync(ctx context.Context) {
	if !m.config.AutoSync {
		return
	}
	
	m.syncTicker = time.NewTicker(m.config.SyncInterval)
	
	go func() {
		log.Printf("Starting auto-sync with interval: %v", m.config.SyncInterval)
		
		for {
			select {
			case <-m.syncTicker.C:
				log.Printf("Running scheduled sync...")
				if err := m.Pull(ctx); err != nil {
					log.Printf("Auto-sync failed: %v", err)
				}
			case <-m.stopChan:
				log.Printf("Stopping auto-sync")
				return
			case <-ctx.Done():
				log.Printf("Context cancelled, stopping auto-sync")
				return
			}
		}
	}()
}

// Stop stops the auto-sync process
func (m *Manager) Stop() {
	if m.syncTicker != nil {
		m.syncTicker.Stop()
	}
	close(m.stopChan)
}

// SetUpdateCallback sets the function to call when files change
func (m *Manager) SetUpdateCallback(callback func(changedFiles []string)) {
	m.onUpdate = callback
}

// GetLastSync returns the time of the last successful sync
func (m *Manager) GetLastSync() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSync
}

// GetLocalPath returns the local repository path
func (m *Manager) GetLocalPath() string {
	return m.config.LocalPath
}

// getAuth returns the authentication method based on config
func (m *Manager) getAuth() transport.AuthMethod {
	// SSH key authentication
	if m.config.SSHKeyPath != "" {
		auth, err := ssh.NewPublicKeysFromFile("git", m.config.SSHKeyPath, "")
		if err == nil {
			return auth
		}
		log.Printf("Failed to load SSH key: %v", err)
	}
	
	// No authentication (for public repos)
	return nil
}

// getFileList returns a map of all files in the repository
func (m *Manager) getFileList() map[string]time.Time {
	files := make(map[string]time.Time)
	
	// Walk the repository directory
	repoPath := m.config.LocalPath
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		
		if !info.IsDir() {
			relPath, _ := filepath.Rel(m.config.LocalPath, path)
			files[relPath] = info.ModTime()
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error walking repository: %v", err)
	}
	
	return files
}

// findChangedFiles compares two file lists and returns changed files
func (m *Manager) findChangedFiles(oldFiles, newFiles map[string]time.Time) []string {
	var changed []string
	
	// Check for new or modified files
	for path, newTime := range newFiles {
		oldTime, exists := oldFiles[path]
		if !exists || !oldTime.Equal(newTime) {
			changed = append(changed, path)
		}
	}
	
	// Check for deleted files
	for path := range oldFiles {
		if _, exists := newFiles[path]; !exists {
			changed = append(changed, path)
		}
	}
	
	return changed
}