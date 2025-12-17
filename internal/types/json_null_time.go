package types

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type JsonNullTime struct {
	sql.NullTime
}

func (j JsonNullTime) String() string {
	if !j.Valid {
		return "NULL"
	}
	return j.Time.Format(MICRO_LAYOUT)
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
	t, err := time.Parse(MICRO_LAYOUT, s)
	if err != nil {
		return JsonNullTime{}, err
	}

	return NewJsonNullTime(t), nil
}

func NewJsonNullTimeStrUnsafe(s string) JsonNullTime {
	t, err := time.Parse(MICRO_LAYOUT, s)
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

// MarshalJSON follows RFC 3339, with microsecond precision, in UTC.
func (j JsonNullTime) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return []byte("null"), nil
	}

	// Encode zero time as JSON null for symmetry with UnmarshalJSON.
	if j.Time.IsZero() {
		return []byte("null"), nil
	}

	// Normalize: strip monotonic, force UTC, and truncate to microseconds.
	t := j.Time.Round(0).UTC().Truncate(time.Microsecond)

	// RFC 3339-compliant string. RFC3339Nano prints fractional seconds only if needed.
	// Because we truncated to µs, at most 6 fractional digits will appear.
	s := t.Format(time.RFC3339Nano)
	return []byte(`"` + s + `"`), nil
}

func (j *JsonNullTime) UnmarshalJSON(data []byte) error {
	if IsNull(data) {
		j.Time = time.Time{}
		j.Valid = false
		return nil // or return an error if "null" is invalid for you
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		j.Valid = false
		return err
	}
	s = strings.TrimSpace(s)
	if s == "" {
		j.Valid = false
		return fmt.Errorf("jsonTime: empty string")
	}

	// 1) Try full RFC3339Nano (handles "…Z" and "…+07:00" with 0–9 fractional digits)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		j.Time = t.UTC() // normalize if you want a consistent zone
		j.Valid = true
		return nil
	}

	// 2) Accept bare timestamp WITHOUT timezone, with optional fractional seconds.
	//    Normalize fractional seconds to exactly 6 digits for parsing.
	//    Examples accepted: "2025-08-26T06:33:59", "…59.9", "…59.9986", "…59.123456789"
	//    Result stored with microsecond precision.
	i := strings.IndexByte(s, '.')
	if i == -1 {
		// No fractional seconds -> add .000000
		s = s + ".000000"
	} else {
		frac := s[i+1:]
		// strip any trailing timezone (shouldn't be present for bare timestamps,
		// but guard anyway)
		if k := strings.IndexAny(frac, "Z+-"); k != -1 {
			// If we find a zone here, it wasn't a bare timestamp, so bail out
			// to a clear error instead of parsing wrong.
			j.Valid = false
			return fmt.Errorf("jsonTime: unexpected timezone in bare timestamp: %q", s)
		}
		switch {
		case len(frac) > 6:
			frac = frac[:6] // truncate to microseconds
		case len(frac) < 6:
			frac = frac + strings.Repeat("0", 6-len(frac)) // right-pad
		}
		s = s[:i] + "." + frac
	}

	// Parse as bare timestamp (no zone).
	t, err := time.Parse(MICRO_LAYOUT, s)
	if err != nil {
		j.Valid = false
		return err
	}
	j.Valid = true
	j.Time = t
	return nil
}
