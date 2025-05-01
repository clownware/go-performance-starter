package repository

import (
	"errors"
)

// Standard repository errors
var (
	// ErrNotFound is returned when a requested entity does not exist
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists is returned when an entity already exists (e.g., when trying to create a duplicate)
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)
