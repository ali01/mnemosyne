package git

import "errors"

var (
	// ErrNoRepoURL indicates that no repository URL was provided in the configuration
	ErrNoRepoURL = errors.New("repository URL is required")
	// ErrNoLocalPath indicates that no local path was provided for the repository clone
	ErrNoLocalPath = errors.New("local path is required")

	// ErrRepoNotFound indicates that the specified repository could not be found
	ErrRepoNotFound   = errors.New("repository not found")
	// ErrBranchNotFound indicates that the specified Git branch could not be found
	ErrBranchNotFound = errors.New("branch not found")
	// ErrCloneFailed indicates that the Git clone operation failed
	ErrCloneFailed    = errors.New("failed to clone repository")
	// ErrPullFailed indicates that the Git pull operation failed
	ErrPullFailed     = errors.New("failed to pull updates")

	// ErrAuthFailed indicates that authentication to the Git repository failed
	ErrAuthFailed = errors.New("authentication failed")

	// ErrSyncInProgress indicates that a sync operation is already running
	ErrSyncInProgress = errors.New("sync already in progress")
)
