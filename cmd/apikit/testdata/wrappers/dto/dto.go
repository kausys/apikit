// Package dto declares a payload outside the handler's package.
package dto

// RenameRequest is a payload from another package of the module.
type RenameRequest struct {
	ID   string     `path:"id"`
	Body RenameBody `json:"body"`
}

// RenameBody is resolved in this package, not the handler's.
type RenameBody struct {
	Name string `json:"name" validate:"required"`
}
