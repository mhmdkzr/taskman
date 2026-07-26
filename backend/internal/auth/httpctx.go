package auth

import (
	"context"
)

type contextKey int

const userContextKey contextKey = iota

type UserInfo struct {
	UserID string
}

func ContextWithUser(ctx context.Context, u UserInfo) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

func UserFromContext(ctx context.Context) (UserInfo, bool) {
	u, ok := ctx.Value(userContextKey).(UserInfo)
	return u, ok
}
