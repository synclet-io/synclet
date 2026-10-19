package app

import (
	"testing"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
)

func TestNotifySMTPDefaultsUseImplicitTLSOn465(t *testing.T) {
	unsetEnv(t, "NOTIFY_SMTP_PORT", "NOTIFY_SMTP_TLS")
	t.Setenv("NOTIFY_SMTP_HOST", "smtp.example.com")

	cfg, err := configutil.NewPrefixedConfigProvider[notifyConfig]("NOTIFY_")()
	require.NoError(t, err)

	assert.Equal(t, 465, cfg.SMTP.Port)
	assert.Equal(t, string(notifyservice.SMTPTLSImplicit), cfg.SMTP.TLS)
}

func TestNotifySMTPEmptyTLSBuildsSender(t *testing.T) {
	cfg := &notifyConfig{SMTP: notifySMTPConfig{Host: "smtp.example.com", Port: 465}}

	sender, err := newSMTPEmailSender(cfg, nil)
	require.NoError(t, err)
	assert.IsType(t, &notifyservice.SMTPEmailSender{}, sender)
}
