package service

import (
	"github.com/ali01/mnemosyne/internal/repository/postgres"
)

// IsNotFound checks if an error is a NotFoundError
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return postgres.IsNotFound(err)
}