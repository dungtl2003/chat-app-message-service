package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type JsonTime struct {
	time.Time
}

func NewJsonTime(t time.Time) JsonTime {
	return JsonTime{t}
}

func NewJsonTimeStrUnsafe(s string) JsonTime {
	layout := "2006-01-02T15:04:05.999Z"
	t, err := time.Parse(layout, s)
	if err != nil {
		return JsonTime{}
	}

	return JsonTime{t}
}

func (j JsonTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Time.Format("2006-01-02T15:04:05.999Z"))
}

func (j *JsonTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return errors.New("jsonTime: UnmarshalJSON(nil)")
	}

	var t string = ""
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

	// we will assume that the time is in UTC
	if t[len(t)-1] != 'Z' {
		t += "Z"
	}

	layout := "2006-01-02T15:04:05.999Z"
	time, err := time.Parse(layout, t)
	if err != nil {
		return err
	}

	j.Time = time
	return nil
}

func (j JsonTime) String() string {
	return j.Time.String()
}

// Scan implements the [Scanner] interface.
func (j *JsonTime) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("jsonTime: Scan(nil)")
	}

	switch value.(type) {
	case time.Time:
		j.Time = value.(time.Time)
		return nil
	default:
		return fmt.Errorf("jsonTime: unsupported type: %T", value)
	}
}

// Value implements the [driver.Valuer] interface.
func (j JsonTime) Value() (driver.Value, error) {
	return j.Time, nil
}
