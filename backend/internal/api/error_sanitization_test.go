package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "git error with sensitive path",
			err:      errors.New("failed to clone git repository from /Users/sensitive/vault: authentication failed"),
			expected: "Failed to sync vault repository",
		},
		{
			name:     "parse error with file path",
			err:      errors.New("parse error in /home/user/vault/secret-note.md: invalid frontmatter"),
			expected: "Failed to parse vault files",
		},
		{
			name:     "database error with connection string",
			err:      errors.New("database connection failed: postgres://user:pass@localhost:5432/db"),
			expected: "Storage operation failed",
		},
		{
			name:     "postgres error",
			err:      errors.New("postgres: connection refused at 192.168.1.100:5432"),
			expected: "Storage operation failed",
		},
		{
			name:     "timeout error",
			err:      errors.New("operation timeout after 30s while processing /var/data/vault"),
			expected: "Operation timed out",
		},
		{
			name:     "parse already in progress",
			err:      errors.New("parse already in progress"),
			expected: "Parse already in progress",
		},
		{
			name:     "panic error with stack trace",
			err:      errors.New("panic during parse: runtime error: index out of range [10] with length 5\ngoroutine 1 [running]:\n..."),
			expected: "An unexpected error occurred during parsing",
		},
		{
			name:     "generic error with system paths",
			err:      errors.New("failed to read file /etc/passwd: permission denied"),
			expected: "Vault processing failed",
		},
		{
			name:     "error with sensitive environment info",
			err:      errors.New("NODE_ENV=production API_KEY=secret123 operation failed"),
			expected: "Vault processing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.err)
			assert.Equal(t, tt.expected, result)

			// Ensure no sensitive information is leaked
			if tt.err != nil {
				assert.NotContains(t, result, "/Users/")
				assert.NotContains(t, result, "/home/")
				assert.NotContains(t, result, "/var/")
				assert.NotContains(t, result, "/etc/")
				assert.NotContains(t, result, "192.168")
				assert.NotContains(t, result, "localhost")
				assert.NotContains(t, result, "postgres://")
				assert.NotContains(t, result, "API_KEY")
				assert.NotContains(t, result, "goroutine")
			}
		})
	}
}

func TestErrorSanitizationIntegration(t *testing.T) {
	// Test that sanitizeParseHistory and sanitizeError work together
	// to prevent information leakage in the complete response

	t.Run("complete error response sanitization", func(t *testing.T) {
		// Simulate various error scenarios and verify sanitized responses
		sensitiveErrors := []error{
			errors.New("git clone failed: /Users/john.doe/private-vault access denied"),
			errors.New("parse failed at line 42 in /var/secrets/config.md"),
			errors.New("postgres connection to 10.0.0.5:5432 timed out"),
		}

		for _, err := range sensitiveErrors {
			sanitized := sanitizeError(err)
			// Verify no sensitive data in sanitized error
			assert.NotContains(t, sanitized, "john.doe")
			assert.NotContains(t, sanitized, "private-vault")
			assert.NotContains(t, sanitized, "/var/secrets")
			assert.NotContains(t, sanitized, "10.0.0.5")
			assert.NotContains(t, sanitized, "line 42")
		}
	})
}
