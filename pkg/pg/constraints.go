package pg

import (
	"errors"

	"github.com/lib/pq"
)

// IsUniqueViolation reports whether the error is a PostgreSQL unique-constraint violation.
func IsUniqueViolation(err error) bool {
	pqErr, ok := errors.AsType[*pq.Error](err)
	return ok && string(pqErr.Code) == "23505"
}
