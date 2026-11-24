package httpapi

import (
	"time"

	"github.com/ozaitsev92/bms/internal/domain"
)

type buildingRequest struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (r buildingRequest) toDomain() domain.Building {
	return domain.Building{ID: r.ID, Name: r.Name, Address: r.Address}
}

type buildingResponse struct {
	ID         int                  `json:"id"`
	Name       string               `json:"name"`
	Address    string               `json:"address"`
	Apartments *[]apartmentResponse `json:"apartments,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

func toBuildingResponse(b domain.Building) buildingResponse {
	resp := buildingResponse{
		ID:        b.ID,
		Name:      b.Name,
		Address:   b.Address,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
	if b.Apartments != nil {
		apartments := toApartmentResponseList(b.Apartments)
		resp.Apartments = &apartments
	}

	return resp
}

func toBuildingResponseList(buildings []domain.Building) []buildingResponse {
	out := make([]buildingResponse, len(buildings))
	for i, b := range buildings {
		out[i] = toBuildingResponse(b)
	}

	return out
}

type apartmentRequest struct {
	ID         int    `json:"id"`
	BuildingID int    `json:"building_id"`
	Number     string `json:"number"`
	Floor      *int   `json:"floor"`
	SqMeters   *int   `json:"sq_meters"`
}

func (r apartmentRequest) toDomain() domain.Apartment {
	a := domain.Apartment{ID: r.ID, BuildingID: r.BuildingID, Number: r.Number}
	if r.Floor != nil {
		a.Floor = *r.Floor
	}
	if r.SqMeters != nil {
		a.SqMeters = *r.SqMeters
	}

	return a
}

func (r apartmentRequest) presenceErr() error {
	if r.Floor == nil {
		return domain.ErrMissingFloor
	}

	return nil
}

type apartmentResponse struct {
	ID         int       `json:"id"`
	BuildingID int       `json:"building_id"`
	Number     string    `json:"number"`
	Floor      int       `json:"floor"`
	SqMeters   int       `json:"sq_meters"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toApartmentResponse(a domain.Apartment) apartmentResponse {
	return apartmentResponse{
		ID:         a.ID,
		BuildingID: a.BuildingID,
		Number:     a.Number,
		Floor:      a.Floor,
		SqMeters:   a.SqMeters,
		CreatedAt:  a.CreatedAt,
		UpdatedAt:  a.UpdatedAt,
	}
}

func toApartmentResponseList(apartments []domain.Apartment) []apartmentResponse {
	out := make([]apartmentResponse, len(apartments))
	for i, a := range apartments {
		out[i] = toApartmentResponse(a)
	}

	return out
}
