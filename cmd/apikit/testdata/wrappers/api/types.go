package api

import (
	"fmt"
	"strings"
	"time"
)

// Status is a string enum: the wrapper casts to it.
type Status string

// ItemID reads itself, like a typed identifier.
type ItemID struct{ value string }

// UnmarshalText accepts "itm_" identifiers only.
func (id *ItemID) UnmarshalText(b []byte) error {
	if !strings.HasPrefix(string(b), "itm_") {
		return fmt.Errorf("not an item id: %q", b)
	}
	id.value = string(b)
	return nil
}

// String returns the identifier.
func (id ItemID) String() string { return id.value }

// ListRequest is declared apart from its handler: the payload resolves from
// any file of the package.
type ListRequest struct {
	Limit  *int      `query:"limit" default:"20"`
	Active *bool     `query:"active"`
	Name   *string   `query:"name"`
	Status Status    `query:"status" default:"active"`
	Kinds  []Status  `query:"kind"`
	Sizes  []int     `query:"size"`
	Since  time.Time `query:"since"`
	Trace  []string  `header:"X-Trace"`
}

// GetRequest binds a typed identifier from the path.
type GetRequest struct {
	ID ItemID `path:"id" validate:"required"`
}

// CreateBody carries the rules: the wrapper validates them although nothing
// above the body has a validate tag.
type CreateBody struct {
	Name  string `json:"name" validate:"required,min=2"`
	Price int    `json:"price"`
}
