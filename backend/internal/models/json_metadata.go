// Package models defines data structures for the application
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMetadata is a custom type for storing JSON data in PostgreSQL
// It automatically handles marshaling/unmarshaling between Go maps and JSON
type JSONMetadata map[string]interface{}

// Scan implements the sql.Scanner interface for database deserialization
func (m *JSONMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("JSONMetadata: cannot scan non-byte value")
	}

	if len(bytes) == 0 {
		*m = nil
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// Value implements the driver.Valuer interface for database serialization
func (m JSONMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// MarshalJSON implements json.Marshaler
func (m JSONMetadata) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return json.Marshal(map[string]interface{}(m))
}

// UnmarshalJSON implements json.Unmarshaler
func (m *JSONMetadata) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*m = nil
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*m = JSONMetadata(result)
	return nil
}