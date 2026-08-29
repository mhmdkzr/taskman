package provision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mhmdkzr/app/internal/identity"
)

func findOrCreate(ctx context.Context, db *sql.DB, subject identity.ZitadelSubject) (identity.UserID, error) {
	userID, err := identity.NewUserID()
	if err != nil {
		return identity.UserID{}, err
	}
	var stored uuid.UUID
	err = db.QueryRowContext(ctx, `
		INSERT INTO users (id, zitadel_sub) VALUES ($1, $2)
		ON CONFLICT (zitadel_sub) DO UPDATE SET zitadel_sub = EXCLUDED.zitadel_sub
		RETURNING id`, userID.UUID(), string(subject)).Scan(&stored)
	if err != nil {
		return identity.UserID{}, fmt.Errorf("upsert user identity mapping: %w", err)
	}
	return identity.UserID(stored), nil
}

func find(ctx context.Context, db *sql.DB, subject identity.ZitadelSubject) (identity.UserID, error) {
	var stored uuid.UUID
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE zitadel_sub = $1`, string(subject)).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.UserID{}, sql.ErrNoRows
	}
	if err != nil {
		return identity.UserID{}, fmt.Errorf("find user identity mapping: %w", err)
	}
	return identity.UserID(stored), nil
}
