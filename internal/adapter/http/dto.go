package http

import "github.com/ozaitsev92/bms/internal/domain"

type buildingRequest struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (r buildingRequest) toDomain() domain.Building {
	return domain.Building{
		ID:      r.ID,
		Name:    r.Name,
		Address: r.Address,
	}
}

type buildingResponse struct {
	ID         int                 `json:"id"`
	Name       string              `json:"name"`
	Address    string              `json:"address"`
	Apartments []apartmentResponse `json:"apartments,omitempty"`
}

func toBuildingResponse(b domain.Building) buildingResponse {
	resp := buildingResponse{
		ID:      b.ID,
		Name:    b.Name,
		Address: b.Address,
	}
	for _, a := range b.Apartments {
		resp.Apartments = append(resp.Apartments, toApartmentResponse(a))
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
	Floor      int    `json:"floor"`
	SqMeters   int    `json:"sq_meters"`
}

func (r apartmentRequest) toDomain() domain.Apartment {
	return domain.Apartment{
		ID:         r.ID,
		BuildingID: r.BuildingID,
		Number:     r.Number,
		Floor:      r.Floor,
		SqMeters:   r.SqMeters,
	}
}

type apartmentResponse struct {
	ID         int    `json:"id"`
	BuildingID int    `json:"building_id"`
	Number     string `json:"number"`
	Floor      int    `json:"floor"`
	SqMeters   int    `json:"sq_meters"`
}

func toApartmentResponse(a domain.Apartment) apartmentResponse {
	return apartmentResponse{
		ID:         a.ID,
		BuildingID: a.BuildingID,
		Number:     a.Number,
		Floor:      a.Floor,
		SqMeters:   a.SqMeters,
	}
}

func toApartmentResponseList(apartments []domain.Apartment) []apartmentResponse {
	out := make([]apartmentResponse, len(apartments))
	for i, a := range apartments {
		out[i] = toApartmentResponse(a)
	}

	return out
}
