// Package provision just-in-time provisions application users from Zitadel subjects.
package provision

import (
	"context"
	"database/sql"

	"github.com/mhmdkzr/app/internal/identity"
)

// FindOrCreate returns the stable app-owned identity mapped to subject.
func FindOrCreate(ctx context.Context, db *sql.DB, subject identity.ZitadelSubject) (identity.UserID, error) {
	return findOrCreate(ctx, db, subject)
}
