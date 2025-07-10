// Package models provides custom database types
package models

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

// StringArray is a custom type for PostgreSQL text[] that handles scanning and valuing
type StringArray []string

// Scan implements the sql.Scanner interface for database deserialization
func (s *StringArray) Scan(src interface{}) error {
	return pq.Array((*[]string)(s)).Scan(src)
}

// Value implements the driver.Valuer interface for database serialization
func (s StringArray) Value() (driver.Value, error) {
	return pq.Array([]string(s)).Value()
}
