package main

import (
	"fmt"
	"github.com/ali01/mnemosyne/internal/models"
)

func main() {
	// Test that all status constants work
	fmt.Println("ParseStatusIdle:", models.ParseStatusIdle)
	fmt.Println("ParseStatusPending:", models.ParseStatusPending)
	fmt.Println("ParseStatusRunning:", models.ParseStatusRunning)
	fmt.Println("ParseStatusCompleted:", models.ParseStatusCompleted)
	fmt.Println("ParseStatusFailed:", models.ParseStatusFailed)

	// Test NewParseStatusFromHistory with nil
	status := models.NewParseStatusFromHistory(nil)
	fmt.Printf("\nStatus for nil history: %s\n", status.Status)

	// Verify it matches the constant
	if status.Status == string(models.ParseStatusIdle) {
		fmt.Println("✓ Status correctly uses ParseStatusIdle constant")
	} else {
		fmt.Printf("✗ Status mismatch: got %s, want %s\n", status.Status, models.ParseStatusIdle)
	}
}
