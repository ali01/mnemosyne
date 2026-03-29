package vault

import (
	"errors"
	"fmt"
)

// Classification errors
var (
	// ErrInvalidRule indicates a classification rule is invalid
	ErrInvalidRule = errors.New("invalid classification rule")
)

// RuleValidationError provides detailed information about rule validation failures
type RuleValidationError struct {
	RuleName string
	Index    int
	Reason   string
}

func (e *RuleValidationError) Error() string {
	if e.RuleName != "" {
		return fmt.Sprintf("invalid rule '%s': %s", e.RuleName, e.Reason)
	}
	return fmt.Sprintf("invalid rule at index %d: %s", e.Index, e.Reason)
}