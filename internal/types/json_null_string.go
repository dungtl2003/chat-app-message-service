package types

import (
	"database/sql"
	"encoding/json"
)

type JsonNullString struct {
	sql.NullString
}

func NewJsonNullString(s string) JsonNullString {
	return JsonNullString{
		sql.NullString{
			String: s,
			Valid:  true,
		},
	}
}

func (j JsonNullString) MarshalJSON() ([]byte, error) {
	if j.Valid {
		return json.Marshal(j.String)
	} else {
		return json.Marshal(nil)
	}
}

func (j *JsonNullString) UnmarshalJSON(data []byte) error {
	var str *string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str != nil {
		j.Valid = true
		j.String = *str
	} else {
		j.Valid = false
	}
	return nil
}
