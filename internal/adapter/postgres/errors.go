// Package postgres implements the repository ports on PostgreSQL using
// sqlboiler-generated models (reads, deletes) and hand-written SQL where
// sqlboiler cannot express the statement (upserts reporting insert vs update).
package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/aarondl/sqlboiler/v4/queries/qm"
	"github.com/lib/pq"

	"github.com/ozaitsev92/bms/internal/domain"
	"github.com/ozaitsev92/bms/internal/port"
)

// PostgreSQL error codes, see https://www.postgresql.org/docs/current/errcodes-appendix.html.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
)

// mapError translates driver errors into the domain errors promised by the
// port contract. Unknown errors are returned unchanged.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("%w: unique constraint %s", domain.ErrConflict, pqErr.Constraint)
		case pgForeignKeyViolation:
			return fmt.Errorf("%w: foreign key %s", domain.ErrInvalidReference, pqErr.Constraint)
		case pgCheckViolation:
			return fmt.Errorf("%w: value violates a database constraint", domain.ErrValidation)
		}
	}

	return err
}

// pageMods converts list parameters into deterministic keyset pagination.
func pageMods(p port.ListParams) []qm.QueryMod {
	mods := []qm.QueryMod{qm.OrderBy("id ASC")}
	if p.AfterID > 0 {
		mods = append(mods, qm.Where("id > ?", p.AfterID))
	}
	if p.Limit > 0 {
		mods = append(mods, qm.Limit(p.Limit))
	}

	return mods
}
