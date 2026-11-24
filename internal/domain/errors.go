package domain

import (
	"errors"
	"fmt"
)

// ErrValidation is wrapped by every input validation error, so callers can
// detect the whole category with errors.Is.
var ErrValidation = errors.New("validation error")

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a write would violate a uniqueness rule,
// for example renaming a building to a name that is already taken.
var ErrConflict = errors.New("conflict")

// ErrInvalidReference is returned when an entity references another entity
// that does not exist, for example an apartment in an unknown building.
var ErrInvalidReference = errors.New("invalid reference")

// Building validation errors.
var (
	ErrMissingBuildingName    = fmt.Errorf("%w: building name is required", ErrValidation)
	ErrMissingBuildingAddress = fmt.Errorf("%w: building address is required", ErrValidation)
)

// Apartment validation errors.
var (
	ErrMissingBuildingID      = fmt.Errorf("%w: building_id must be a positive integer", ErrValidation)
	ErrMissingApartmentNumber = fmt.Errorf("%w: apartment number is required", ErrValidation)
	ErrMissingFloor           = fmt.Errorf("%w: floor is required", ErrValidation)
	ErrInvalidSqMeters        = fmt.Errorf("%w: sq_meters must be greater than zero", ErrValidation)
)
