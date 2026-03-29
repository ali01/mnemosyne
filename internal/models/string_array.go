// Package models provides custom database types
package models

import (
	"database/sql/driver"
	"encoding/json"
)

// StringArray is a custom type that serializes as a JSON array in SQLite.
type StringArray []string

// Scan implements the sql.Scanner interface for database deserialization.
func (s *StringArray) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		*s = nil
		return nil
	}
	return json.Unmarshal(data, s)
}

// Value implements the driver.Valuer interface for database serialization.
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}
