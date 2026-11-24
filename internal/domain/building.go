// Package domain holds the core business entities, their invariants and the
// errors that form the contract between the application and its adapters.
package domain

import (
	"errors"
	"strings"
	"time"
)

// Building represents building data in the business domain.
type Building struct {
	ID         int
	Name       string
	Address    string
	Apartments []Apartment
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Validate checks the building's invariants and reports every violation.
func (b Building) Validate() error {
	var errs []error
	if strings.TrimSpace(b.Name) == "" {
		errs = append(errs, ErrMissingBuildingName)
	}
	if strings.TrimSpace(b.Address) == "" {
		errs = append(errs, ErrMissingBuildingAddress)
	}

	return errors.Join(errs...)
}
