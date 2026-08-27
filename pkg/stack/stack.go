// Package stack anchors third-party stack dependencies so `go mod tidy` does not prune them.
//
// Zitadel, TigerBeetle, and MailHog are part of the stack (see compose.yaml).
// Their Go clients are intentionally retained even when no slice imports them yet.
// This file intentionally uses blank imports to keep the modules in go.mod/go.sum.
package stack

import (
	_ "github.com/tigerbeetle/tigerbeetle-go"
	_ "github.com/wneessen/go-mail"
	_ "github.com/zitadel/zitadel-go/v3/pkg/client"
)
