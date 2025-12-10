// Package ast provides generic AST parsing capabilities for Go source files.
// This package is completely agnostic to any specific directives (apikit, swagger, etc.)
// and provides raw AST information that can be consumed by different adapters.
package ast

import (
	"go/ast"
	"go/token"
)

// ParseResult contains the raw AST information from a parsed file
type ParseResult struct {
	// File is the parsed AST file
	File *ast.File

	// Structs contains all struct definitions found in the file
	Structs map[string]*Struct

	// Functions contains all function/method declarations found in the file
	Functions []*Function

	// Imports maps import aliases to their full import paths
	Imports map[string]string

	// Package is the package name
	Package string

	// Filename is the source file path
	Filename string

	// FileSet is the token file set for position information
	FileSet *token.FileSet
}

// Struct represents a struct type with all its information
type Struct struct {
	// Name is the struct name
	Name string

	// TypeSpec is the raw AST type spec node
	TypeSpec *ast.TypeSpec

	// StructType is the raw AST struct type node
	StructType *ast.StructType

	// Doc contains documentation comments above the struct
	Doc *ast.CommentGroup

	// Comment contains comments on the same line as the struct
	Comment *ast.CommentGroup

	// Fields contains all struct fields
	Fields []*Field

	// Position in source file
	Pos token.Position
}

// Field represents a struct field with all its information
type Field struct {
	// Name is the field name (empty for embedded fields)
	Name string

	// Type is the Go type as string (e.g., "string", "*int", "[]string", "pkg.Type")
	Type string

	// ASTField is the raw AST field node
	ASTField *ast.Field

	// ASTType is the raw AST type expression
	ASTType ast.Expr

	// Doc contains documentation comments above the field
	Doc *ast.CommentGroup

	// Comment contains comments on the same line as the field
	Comment *ast.CommentGroup

	// Tag is the struct tag string (without backticks)
	Tag string

	// Type information
	IsPointer bool   // Is this a pointer type (*string)
	IsSlice   bool   // Is this a slice type ([]string)
	SliceType string // Element type for slices (e.g., "string" for []string)

	// Embedded field
	IsEmbedded bool

	// Position in source file
	Pos token.Position
}

// Function represents a function or method declaration
type Function struct {
	// Name is the function name
	Name string

	// FuncDecl is the raw AST function declaration node
	FuncDecl *ast.FuncDecl

	// Doc contains documentation comments above the function
	Doc *ast.CommentGroup

	// Comment contains comments on the same line as the function
	Comment *ast.CommentGroup

	// Receiver is the receiver type for methods (empty for functions)
	Receiver string

	// ReceiverType is the raw AST receiver type expression
	ReceiverType ast.Expr

	// Params contains function parameters
	Params []*Param

	// Results contains function return values
	Results []*Param

	// Position in source file
	Pos token.Position
}

// Param represents a function parameter or result
type Param struct {
	// Name is the parameter name (can be empty)
	Name string

	// Type is the Go type as string
	Type string

	// ASTType is the raw AST type expression
	ASTType ast.Expr

	// IsPointer indicates if this is a pointer type
	IsPointer bool

	// IsSlice indicates if this is a slice type
	IsSlice bool

	// IsVariadic indicates if this is a variadic parameter
	IsVariadic bool
}

// Import represents an import declaration
type Import struct {
	// Alias is the import alias (empty if no alias)
	Alias string

	// Path is the import path
	Path string

	// Doc contains documentation comments above the import
	Doc *ast.CommentGroup

	// Comment contains comments on the same line as the import
	Comment *ast.CommentGroup
}

