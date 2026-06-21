package domain

// Apartment represents apartment data in the business domain.
type Apartment struct {
	ID         int
	BuildingID int
	Number     string
	Floor      int
	SqMeters   int
}
