package types

import (
	"database/sql/driver"
	"encoding/json"
)

type Json struct {
	json.RawMessage
}

func NewJson(data any) Json {
	if data == nil {
		return Json{}
	}
	if len(data.([]byte)) == 0 {
		return Json{RawMessage: json.RawMessage{}}
	}
	return Json{RawMessage: data.([]byte)}
}

func NewJsonFromString(s string) Json {
	if s == "" {
		return Json{}
	}
	if s == "{}" {
		return Json{RawMessage: json.RawMessage{}}
	}
	return Json{RawMessage: json.RawMessage(s)}
}

func (j Json) String() string {
	if j.RawMessage == nil {
		return ""
	}
	if len(j.RawMessage) == 0 {
		return "{}"
	}
	return string(j.RawMessage)
}

func (j *Json) Scan(value any) error {
	if value == nil {
		j.RawMessage = nil
		return nil
	}

	switch value := value.(type) {
	case []byte:
		// Create a deep copy of the bytes
		j.RawMessage = make([]byte, len(value))
		copy(j.RawMessage, value)
		return nil
	case string:
		j.RawMessage = []byte(value)
		return nil
	default:
		return nil // Or return an error if you want to be strict
	}
}

func (j Json) Value() (driver.Value, error) {
	if j.RawMessage == nil {
		return nil, nil
	}
	if len(j.RawMessage) == 0 {
		// Return empty JSON object {} for empty slice to satisfy JSONB columns
		return []byte("{}"), nil
	}

	return []byte(j.RawMessage), nil
}

func (j Json) MarshalJSON() ([]byte, error) {
	if j.RawMessage == nil {
		return []byte("null"), nil
	}
	if len(j.RawMessage) == 0 {
		return []byte("{}"), nil
	}
	return j.RawMessage, nil
}

func UnmarshalJSON(data Json, target any) error {
	return json.Unmarshal(data.RawMessage, target)
}
