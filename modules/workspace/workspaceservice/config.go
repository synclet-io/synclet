package workspaceservice

import "time"

// Config holds workspace service configuration passed from the app layer.
type Config struct {
	InviteTTL   time.Duration
	FrontendURL string
}
