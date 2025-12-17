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

func (j Json) String() string {
	if j.RawMessage == nil {
		return ""
	}
	if len(j.RawMessage) == 0 {
		return "{}"
	}
	return string(j.RawMessage)
}

func (j *Json) Dereference() Json {
	if j == nil {
		return Json{}
	}
	return *j
}

func (j *Json) Scan(value any) error {
	if value == nil {
		j.RawMessage = nil
		return nil
	}

	switch value := value.(type) {
	case []byte:
		j.RawMessage = value
		return nil
	case string:
		j.RawMessage = []byte(value)
		return nil
	default:
		return nil
	}
}

func (j Json) Value() (driver.Value, error) {
	if j.RawMessage == nil {
		return nil, nil
	}
	if len(j.RawMessage) == 0 {
		return []byte{}, nil
	}

	rawBytes, err := json.Marshal(j.RawMessage)
	if err != nil {
		return nil, err
	}
	return rawBytes, nil
}
