package domain

// Building represents building data in the business domain.
type Building struct {
	ID         int
	Name       string
	Address    string
	Apartments []Apartment
}
