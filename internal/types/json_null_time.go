package types

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// Time must have the format "2006-01-02T15:04:05.999Z" (always in UTC). If missing Z, we will add Z to the end of the string.
type JsonNullTime struct {
	sql.NullTime
}

func (j JsonNullTime) String() string {
	return j.Time.String()
}

func (j JsonNullTime) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(j.Time)
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		j.Valid = false
		return nil
	}

	var t string = ""
	if err := json.Unmarshal(data, &t); err != nil {
		log.Println("Error when unmarshal json null time")
		return err
	}
	if t != "" {
		// we will assume that the time is in UTC
		if t[len(t)-1] != 'Z' {
			t += "Z"
		}

		layout := "2006-01-02T15:04:05.999Z"
		time, err := time.Parse(layout, t)
		if err != nil {
			log.Println("Error when parsing time:", err)
			j.Valid = false
			return err
		}

		j.Valid = true
		j.Time = time
	} else {
		j.Valid = false
	}
	return nil
}

func DefaultJsonNullTime() JsonNullTime {
	return JsonNullTime{
		sql.NullTime{
			Valid: false,
		},
	}
}

func NewJsonNullTime(t time.Time) JsonNullTime {
	return JsonNullTime{
		sql.NullTime{
			Time:  t,
			Valid: true,
		},
	}
}

func NewJsonNullTimeStr(s string) (JsonNullTime, error) {
	layout := "2006-01-02T15:04:05.999Z"
	t, err := time.Parse(layout, s)
	if err != nil {
		return JsonNullTime{}, err
	}

	return NewJsonNullTime(t), nil
}

func NewJsonNullTimeStrUnsafe(s string) JsonNullTime {
	layout := "2006-01-02T15:04:05.999Z"
	t, err := time.Parse(layout, s)
	if err != nil {
		return JsonNullTime{
			sql.NullTime{
				Valid: false,
			},
		}
	}

	return JsonNullTime{
		sql.NullTime{
			Time:  t,
			Valid: true,
		},
	}
}
