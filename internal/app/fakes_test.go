package app_test

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

// The fakes honour the port error contract exactly like the Postgres adapter:
// missing rows -> domain.ErrNotFound, duplicate keys -> domain.ErrConflict,
// unknown building -> domain.ErrInvalidReference. Setting err forces every
// method to fail, to test propagation.

var errBoom = errors.New("boom")

type fakeBuildingRepo struct {
	mu     sync.Mutex
	rows   map[int]domain.Building
	nextID int
	err    error
}

func newFakeBuildingRepo() *fakeBuildingRepo {
	return &fakeBuildingRepo{rows: map[int]domain.Building{}, nextID: 1}
}

func (f *fakeBuildingRepo) FindAll(_ context.Context, p port.ListParams, _ bool) ([]domain.Building, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	out := make([]domain.Building, 0, len(f.rows))
	for _, b := range f.rows {
		if b.ID > p.AfterID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if p.Limit > 0 && len(out) > p.Limit {
		out = out[:p.Limit]
	}
	return out, nil
}

func (f *fakeBuildingRepo) FindByID(_ context.Context, id int, _ bool) (*domain.Building, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.rows[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &b, nil
}

func (f *fakeBuildingRepo) Upsert(_ context.Context, b domain.Building) (*domain.Building, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, false, f.err
	}
	for id, existing := range f.rows {
		if existing.Name == b.Name && id != b.ID {
			if b.ID != 0 {
				return nil, false, domain.ErrConflict
			}
			existing.Address = b.Address
			f.rows[id] = existing
			return &existing, false, nil
		}
	}
	if b.ID != 0 {
		if _, ok := f.rows[b.ID]; !ok {
			return nil, false, domain.ErrNotFound
		}
		f.rows[b.ID] = b
		return &b, false, nil
	}
	b.ID = f.nextID
	f.nextID++
	f.rows[b.ID] = b
	return &b, true, nil
}

func (f *fakeBuildingRepo) Delete(_ context.Context, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if _, ok := f.rows[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.rows, id)
	return nil
}

type fakeApartmentRepo struct {
	mu        sync.Mutex
	rows      map[int]domain.Apartment
	buildings map[int]bool
	nextID    int
	err       error
}

func newFakeApartmentRepo(buildingIDs ...int) *fakeApartmentRepo {
	f := &fakeApartmentRepo{rows: map[int]domain.Apartment{}, buildings: map[int]bool{}, nextID: 1}
	for _, id := range buildingIDs {
		f.buildings[id] = true
	}
	return f
}

func (f *fakeApartmentRepo) list(p port.ListParams, keep func(domain.Apartment) bool) []domain.Apartment {
	out := make([]domain.Apartment, 0, len(f.rows))
	for _, a := range f.rows {
		if a.ID > p.AfterID && keep(a) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if p.Limit > 0 && len(out) > p.Limit {
		out = out[:p.Limit]
	}
	return out
}

func (f *fakeApartmentRepo) FindAll(_ context.Context, p port.ListParams) ([]domain.Apartment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return f.list(p, func(domain.Apartment) bool { return true }), nil
}

func (f *fakeApartmentRepo) FindByID(_ context.Context, id int) (*domain.Apartment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	a, ok := f.rows[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &a, nil
}

func (f *fakeApartmentRepo) FindByBuildingID(_ context.Context, buildingID int, p port.ListParams) ([]domain.Apartment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if !f.buildings[buildingID] {
		return nil, domain.ErrNotFound
	}
	return f.list(p, func(a domain.Apartment) bool { return a.BuildingID == buildingID }), nil
}

func (f *fakeApartmentRepo) Upsert(_ context.Context, a domain.Apartment) (*domain.Apartment, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, false, f.err
	}
	if !f.buildings[a.BuildingID] {
		return nil, false, domain.ErrInvalidReference
	}
	for id, existing := range f.rows {
		if existing.BuildingID == a.BuildingID && existing.Number == a.Number && id != a.ID {
			if a.ID != 0 {
				return nil, false, domain.ErrConflict
			}
			a.ID = id
			f.rows[id] = a
			return &a, false, nil
		}
	}
	if a.ID != 0 {
		if _, ok := f.rows[a.ID]; !ok {
			return nil, false, domain.ErrNotFound
		}
		f.rows[a.ID] = a
		return &a, false, nil
	}
	a.ID = f.nextID
	f.nextID++
	f.rows[a.ID] = a
	return &a, true, nil
}

func (f *fakeApartmentRepo) Delete(_ context.Context, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if _, ok := f.rows[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.rows, id)
	return nil
}
