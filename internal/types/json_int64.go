package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

// When we parse number from JSON, it will be parsed as string (to prevent wrong number when parsing big number)
type JsonInt64 struct {
	int64
}

func NewJsonInt64(i int64) JsonInt64 {
	return JsonInt64{i}
}

func (j JsonInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(j.int64, 10))
}

func (j *JsonInt64) UnmarshalJSON(data []byte) error {
	// data can be string or int64
	var i int64 = 0
	var str string = ""

	if err := json.Unmarshal(data, &str); err != nil {
		if err := json.Unmarshal(data, &i); err != nil {
			return err
		}
	}

	if str != "" {
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		j.int64 = i
	} else if i != 0 {
		j.int64 = i
	}

	return nil
}

// Scan implements the [Scanner] interface.
func (j *JsonInt64) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("jsonInt64: Scan(nil)")
	}

	switch value := value.(type) {
	case int64:
		j.int64 = value
		return nil
	default:
		return fmt.Errorf("jsonInt64: unsupported type: %T", value)
	}
}

// Value implements the [driver.Valuer] interface.
func (j JsonInt64) Value() (driver.Value, error) {
	return j.int64, nil
}

func (j JsonInt64) Int64() int64 {
	return j.int64
}

func (j JsonInt64) String() string {
	return strconv.FormatInt(j.int64, 10)
}
