package workspaceservice

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// newInviteToken returns a random invite token and the SHA-256 hash stored in WorkspaceInvite.Token.
func newInviteToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating invite token: %w", err)
	}

	raw = hex.EncodeToString(b)

	return raw, hashInviteToken(raw), nil
}

func hashInviteToken(raw string) string {
	h := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(h[:])
}
