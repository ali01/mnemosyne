package git

import "errors"

var (
	// Configuration errors
	ErrNoRepoURL   = errors.New("repository URL is required")
	ErrNoLocalPath = errors.New("local path is required")
	
	// Repository errors
	ErrRepoNotFound   = errors.New("repository not found")
	ErrBranchNotFound = errors.New("branch not found")
	ErrCloneFailed    = errors.New("failed to clone repository")
	ErrPullFailed     = errors.New("failed to pull updates")
	
	// Authentication errors
	ErrAuthFailed = errors.New("authentication failed")
	
	// Sync errors
	ErrSyncInProgress = errors.New("sync already in progress")
)