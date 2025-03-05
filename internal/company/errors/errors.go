package errors

import (
	"fmt"
)

var (
	ErrNotFound      = fmt.Errorf("not found")
	ErrDuplicateName = fmt.Errorf("duplicate name")
	ErrInvalidInput  = fmt.Errorf("invalid input")
)
