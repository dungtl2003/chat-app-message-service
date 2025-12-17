package types

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
)

type JsonNullInt64 struct {
	sql.NullInt64
}

func (j JsonNullInt64) String() string {
	if j.Valid {
		return strconv.FormatInt(j.Int64, 10)
	}
	return "NULL"
}

func (j JsonNullInt64) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(strconv.FormatInt(j.Int64, 10))
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullInt64) UnmarshalJSON(data []byte) error {
	if IsNull(data) {
		j.Valid = false
		return nil
	}

	var i int64 = 0
	var str string = ""

	if err := json.Unmarshal(data, &str); err != nil {
		if err := json.Unmarshal(data, &i); err != nil {
			return err
		}
	}

	j.Valid = true
	if str != "" {
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		j.Int64 = i
	} else if i != 0 {
		j.Int64 = i
	}

	return nil
}

func (j JsonNullInt64) ToJsonInt64() JsonInt64 {
	if j.Valid {
		return NewJsonInt64(j.Int64)
	}
	return NewJsonInt64(0)
}

// Scan implements the [Scanner] interface.
func (j *JsonNullInt64) Scan(value any) error {
	if value == nil {
		j.Valid = false
		j.Int64 = 0
		return nil
	}

	switch value := value.(type) {
	case int64:
		j.Valid = true
		j.Int64 = value
		return nil
	default:
		return fmt.Errorf("jsonNullInt64: unsupported type: %T", value)
	}
}

// Value implements the [driver.Valuer] interface.
func (j JsonNullInt64) Value() (driver.Value, error) {
	if !j.Valid {
		return nil, nil // Return nil for null value
	}

	return j.Int64, nil
}

func NewJsonNullInt64(i int64) JsonNullInt64 {
	return JsonNullInt64{
		sql.NullInt64{
			Int64: i,
			Valid: true,
		},
	}
}

func NewJsonNullInt64Null() JsonNullInt64 {
	return JsonNullInt64{
		sql.NullInt64{
			Int64: 0,
			Valid: false,
		},
	}
}
