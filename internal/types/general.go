package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type OutboxStatus string

const (
	MICRO_LAYOUT   string       = "2006-01-02T15:04:05.000000"
	OUTBOX_PENDING OutboxStatus = "PENDING"
	OUTBOX_SENT    OutboxStatus = "SENT"
)

type ErrorItem struct {
	Message string `json:"message" validate:"omitempty"`
	Reason  string `json:"reason" validate:"omitempty"` // e.g. INVALID_DEVICE, EMAIL_EXISTS, etc
	Domain  string `json:"domain" validate:"omitempty"`
}

type ErrorBlock struct {
	Code    int    `json:"code" validate:"omitempty,numeric"`
	Message string `json:"message" validate:"omitempty"`
	// this is the same as Reason in ErrorItem, but for primary error for client to switch on
	Status string      `json:"status" validate:"omitempty"` // e.g. "INVALID_ARGUMENT"
	Errors []ErrorItem `json:"errors" validate:"omitempty"`
}

// Optional paging & collection metadata that can appear on list responses
// Any int field will be replaced with JsonNullInt64 to support 64-bit values
type PageInfo struct {
	Kind    string `json:"kind" validate:"omitempty"`
	Fields  string `json:"fields" validate:"omitempty"`
	ETag    string `json:"etag" validate:"omitempty"`
	Lang    string `json:"lang" validate:"omitempty"`
	Updated string `json:"updated" validate:"omitempty"` // RFC 3339 string
	Deleted bool   `json:"deleted" validate:"omitempty"`
	// Equivalent to items.length in JS
	CurrentItemCount JsonNullInt64 `json:"current_item_count" validate:"omitempty"`
	// items.length <= items_per_page
	ItemsPerPage JsonNullInt64 `json:"items_per_page" validate:"omitempty"`
	TotalItems   JsonNullInt64 `json:"total_items" validate:"omitempty"`
	// total_pages = ceil(total_items / items_per_page)
	TotalPages JsonNullInt64 `json:"total_pages" validate:"omitempty"`

	// Some APIs expose both object links and plain URL links; keep them flexible.
	Next         json.RawMessage `json:"next" validate:"omitempty"`
	NextLink     string          `json:"next_link" validate:"omitempty"`
	Previous     json.RawMessage `json:"previous" validate:"omitempty"`
	PreviousLink string          `json:"previous_link" validate:"omitempty"`
	Self         json.RawMessage `json:"self" validate:"omitempty"`
	SelfLink     string          `json:"self_link" validate:"omitempty"`
	Edit         json.RawMessage `json:"edit" validate:"omitempty"`
	EditLink     string          `json:"edit_link" validate:"omitempty"`

	// Cursor-based pagination
	EndCursor string `json:"end_cursor" validate:"omitempty"`
	// If hasMore appears, EndCursor should be true
	HasMore bool `json:"has_more" validate:"omitempty"`
}

type Page[T any] struct {
	PageInfo
	Items []T `json:"items"`
}

// DataOrPage[T] can hold either a single T or a Page[T].
// It also keeps Raw for debugging/logging if needed.
type DataOrPage[T any] struct {
	Item *T
	Page *Page[T]
	Raw  json.RawMessage
}

// Follow the structure of Google API response
type Response[T any] struct {
	ApiVersion string          `json:"api_version" validate:"omitempty"`
	Method     string          `json:"method" validate:"omitempty"`
	Params     json.RawMessage `json:"params" validate:"omitempty"`

	Error *ErrorBlock    `json:"error" validate:"omitempty"`
	Data  *DataOrPage[T] `json:"data" validate:"omitempty"`

	// References holds side-loaded data (users, files, etc.)
	References map[string]any `json:"references,omitempty"`
}

func (r Response[T]) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		// Fallback: don’t panic in Stringer
		return fmt.Sprintf("Response{error: %v}", err)
	}
	return string(b)
}

func (d DataOrPage[T]) MarshalJSON() ([]byte, error) {
	switch {
	case d.Page != nil:
		return json.Marshal(d.Page)
	case d.Item != nil:
		return json.Marshal(d.Item)
	default:
		// if we kept Raw (e.g., round-trip), return it; else empty object
		if len(d.Raw) > 0 && bytes.HasPrefix(d.Raw, []byte("{")) {
			return d.Raw, nil
		}
		return json.Marshal(nil)
	}
}

func (e ErrorBlock) MarshalJSON() ([]byte, error) {
	// If Code is zero and Message is empty, marshal as null
	if e.Code == 0 && e.Message == "" && len(e.Errors) == 0 {
		return json.Marshal(nil)
	}

	type Alias ErrorBlock // Prevent recursion
	return json.Marshal((Alias)(e))
}

// UnmarshalJSON tries to detect whether the payload is a Page (has "items")
// or a single resource (T).
func (d *DataOrPage[T]) UnmarshalJSON(b []byte) error {
	d.Raw = append(d.Raw[:0], b...)

	// Quick probe: does it contain "items" at the top level?
	var probe struct {
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(b, &probe); err == nil && len(probe.Items) > 0 {
		var p Page[T]
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		d.Page, d.Item = &p, nil
		return nil
	}

	// Otherwise decode as a single resource T
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d.Item, d.Page = &v, nil
	return nil
}

// IsNull returns true if the given JSON data represents a null value,
func IsNull(data []byte) bool {
	return string(data) == "null" || len(data) == 0 || string(data) == `""`
}
