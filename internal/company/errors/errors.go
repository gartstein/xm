package errors

import (
	"fmt"
)

var (
	ErrCompanyIDRequired = fmt.Errorf("company ID required")
	ErrInvalidCompanyID  = fmt.Errorf("invalid company ID format")
	ErrNotFound          = fmt.Errorf("not found")
	ErrDuplicateName     = fmt.Errorf("duplicate name")
	ErrInvalidInput      = fmt.Errorf("invalid input")
)
