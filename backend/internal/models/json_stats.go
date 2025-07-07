// Package models defines data structures for the application
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONStats is a custom type for storing ParseStats as JSON in PostgreSQL
type JSONStats ParseStats

// Scan implements the sql.Scanner interface for database deserialization
func (s *JSONStats) Scan(value interface{}) error {
	if value == nil {
		*s = JSONStats{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("JSONStats: cannot scan non-byte value")
	}

	if len(bytes) == 0 {
		*s = JSONStats{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for database serialization
func (s JSONStats) Value() (driver.Value, error) {
	if s == (JSONStats{}) {
		return nil, nil
	}
	return json.Marshal(s)
}

// MarshalJSON implements json.Marshaler
func (s JSONStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(ParseStats(s))
}

// UnmarshalJSON implements json.Unmarshaler
func (s *JSONStats) UnmarshalJSON(data []byte) error {
	var stats ParseStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return err
	}
	*s = JSONStats(stats)
	return nil
}

// ToParseStats converts JSONStats back to ParseStats
func (s JSONStats) ToParseStats() ParseStats {
	return ParseStats(s)
}