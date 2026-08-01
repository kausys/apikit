package textunmarshaler

import "context"

// TypedID implements encoding.TextUnmarshaler (struct, not string alias).
type TypedID struct {
	raw string
}

func (id *TypedID) UnmarshalText(text []byte) error {
	id.raw = string(text)
	return nil
}

// Status is a string enum — cast fallback, not TextUnmarshaler.
type Status string

type GetThingPayload struct {
	// in:path
	ID TypedID `json:"id"`

	// in:query
	Filter Status `json:"filter"`

	// in:query
	OptionalID *TypedID `json:"optionalId"`
}

type GetThingResponse struct {
	OK bool `json:"ok"`
}

// apikit:handler
func GetThing(ctx context.Context, req GetThingPayload) (GetThingResponse, error) {
	return GetThingResponse{OK: true}, nil
}
