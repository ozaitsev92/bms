package app

import (
	"errors"
	"fmt"
)

// sentinel that handler can check without knowing every error variant
var ErrValidation = errors.New("validation error")

// Building errors
var (
	ErrMissingBuildingName    = fmt.Errorf("%w: building name is required", ErrValidation)
	ErrMissingBuildingAddress = fmt.Errorf("%w: building address is required", ErrValidation)
)

// Apartment errors
var (
	ErrMissingBuildingID      = fmt.Errorf("%w: building_id is required", ErrValidation)
	ErrMissingApartmentNumber = fmt.Errorf("%w: apartment number is required", ErrValidation)
	ErrInvalidFloor           = fmt.Errorf("%w: floor must be non-zero", ErrValidation)
	ErrInvalidSqMeters        = fmt.Errorf("%w: sq_meters must be greater than zero", ErrValidation)
)

// Shared
var ErrNotFound = errors.New("not found")
