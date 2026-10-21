package authstorage

import (
	fmt "fmt"

	authservice "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

const (
	oAuthAccessLevelRead    = "read"
	oAuthAccessLevelActions = "actions"
	oAuthAccessLevelWrite   = "write"
)

func convertOAuthAccessLevelToDB(oAuthAccessLevelValue authservice.OAuthAccessLevel) (string, error) {
	result, ok := map[authservice.OAuthAccessLevel]string{
		authservice.OAuthAccessLevelRead:    oAuthAccessLevelRead,
		authservice.OAuthAccessLevelActions: oAuthAccessLevelActions,
		authservice.OAuthAccessLevelWrite:   oAuthAccessLevelWrite,
	}[oAuthAccessLevelValue]
	if !ok {
		return "", fmt.Errorf("unknown OAuthAccessLevel value: %d", oAuthAccessLevelValue)
	}
	return result, nil
}

func convertOAuthAccessLevelFromDB(oAuthAccessLevelValue string) (authservice.OAuthAccessLevel, error) {
	result, ok := map[string]authservice.OAuthAccessLevel{
		oAuthAccessLevelRead:    authservice.OAuthAccessLevelRead,
		oAuthAccessLevelActions: authservice.OAuthAccessLevelActions,
		oAuthAccessLevelWrite:   authservice.OAuthAccessLevelWrite,
	}[oAuthAccessLevelValue]
	if !ok {
		return 0, fmt.Errorf("unknown OAuthAccessLevel db value: %s", oAuthAccessLevelValue)
	}
	return result, nil
}
