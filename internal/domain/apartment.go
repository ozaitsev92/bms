package domain

import (
	"errors"
	"strings"
	"time"
)

// Apartment represents apartment data in the business domain.
//
// Floor may be zero (ground floor) or negative (basement levels); whether a
// floor was supplied at all is checked at the transport boundary.
type Apartment struct {
	ID         int
	BuildingID int
	Number     string
	Floor      int
	SqMeters   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate checks the apartment's invariants and reports every violation.
func (a Apartment) Validate() error {
	var errs []error
	if a.BuildingID <= 0 {
		errs = append(errs, ErrMissingBuildingID)
	}
	if strings.TrimSpace(a.Number) == "" {
		errs = append(errs, ErrMissingApartmentNumber)
	}
	if a.SqMeters <= 0 {
		errs = append(errs, ErrInvalidSqMeters)
	}

	return errors.Join(errs...)
}
