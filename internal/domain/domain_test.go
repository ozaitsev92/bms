package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ozaitsev92/bms/internal/domain"
)

func TestBuilding_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      domain.Building
		wantErr []error
	}{
		{name: "valid", in: domain.Building{Name: "Tower", Address: "1 St"}},
		{name: "missing name", in: domain.Building{Address: "1 St"}, wantErr: []error{domain.ErrMissingBuildingName}},
		{name: "whitespace name", in: domain.Building{Name: "   ", Address: "1 St"}, wantErr: []error{domain.ErrMissingBuildingName}},
		{name: "missing address", in: domain.Building{Name: "Tower"}, wantErr: []error{domain.ErrMissingBuildingAddress}},
		{
			name:    "reports all violations",
			in:      domain.Building{},
			wantErr: []error{domain.ErrMissingBuildingName, domain.ErrMissingBuildingAddress},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.in.Validate()
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, domain.ErrValidation)
			for _, want := range tt.wantErr {
				assert.ErrorIs(t, err, want)
			}
		})
	}
}

func TestApartment_Validate(t *testing.T) {
	t.Parallel()

	valid := domain.Apartment{BuildingID: 1, Number: "101", Floor: 1, SqMeters: 50}

	tests := []struct {
		name    string
		mutate  func(a *domain.Apartment)
		wantErr []error
	}{
		{name: "valid", mutate: func(*domain.Apartment) {}},
		{name: "ground floor accepted", mutate: func(a *domain.Apartment) { a.Floor = 0 }},
		{name: "basement accepted", mutate: func(a *domain.Apartment) { a.Floor = -2 }},
		{name: "missing building", mutate: func(a *domain.Apartment) { a.BuildingID = 0 }, wantErr: []error{domain.ErrMissingBuildingID}},
		{name: "negative building", mutate: func(a *domain.Apartment) { a.BuildingID = -1 }, wantErr: []error{domain.ErrMissingBuildingID}},
		{name: "whitespace number", mutate: func(a *domain.Apartment) { a.Number = " " }, wantErr: []error{domain.ErrMissingApartmentNumber}},
		{name: "zero sq meters", mutate: func(a *domain.Apartment) { a.SqMeters = 0 }, wantErr: []error{domain.ErrInvalidSqMeters}},
		{name: "negative sq meters", mutate: func(a *domain.Apartment) { a.SqMeters = -5 }, wantErr: []error{domain.ErrInvalidSqMeters}},
		{
			name:    "reports all violations",
			mutate:  func(a *domain.Apartment) { *a = domain.Apartment{} },
			wantErr: []error{domain.ErrMissingBuildingID, domain.ErrMissingApartmentNumber, domain.ErrInvalidSqMeters},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := valid
			tt.mutate(&a)
			err := a.Validate()
			if len(tt.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, domain.ErrValidation)
			for _, want := range tt.wantErr {
				assert.ErrorIs(t, err, want)
			}
		})
	}
}
