package connectutil

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type contextKey int

const (
	userIDKey contextKey = iota
	emailKey
	workspaceIDKey
	clientIPKey
)

// ContextWithUserID returns a context with the user ID set.
func ContextWithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// ContextWithEmail returns a context with the email set.
func ContextWithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey, email)
}

// ContextWithWorkspaceID returns a context with the workspace ID set.
func ContextWithWorkspaceID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, workspaceIDKey, id)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("user ID not found in context")
	}

	return v, nil
}

// EmailFromContext extracts the email from the context.
func EmailFromContext(ctx context.Context) (string, error) {
	v, ok := ctx.Value(emailKey).(string)
	if !ok || v == "" {
		return "", errors.New("email not found in context")
	}

	return v, nil
}

// WorkspaceIDFromContext extracts the workspace ID from the context.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(workspaceIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("workspace ID not found in context")
	}

	return v, nil
}

// ContextWithClientIP returns a context with the client IP of the request set.
func ContextWithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// ClientIPFromContext extracts the client IP of the request from the context.
func ClientIPFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(clientIPKey).(string)

	return v, ok && v != ""
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
