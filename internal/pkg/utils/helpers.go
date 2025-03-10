// Package utils provides common utility functions.
package utils

// Ptr returns a pointer to the given value.
func Ptr[T any](value T) *T {
	return &value
}
