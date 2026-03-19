package notifyservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/multierr"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

// TelegramChannel delivers notifications via the Telegram Bot API.
type TelegramChannel struct {
	secrets    SecretsProvider
	httpClient *http.Client
}

// NewTelegramChannel creates a new TelegramChannel deliverer.
func NewTelegramChannel(secrets SecretsProvider) *TelegramChannel {
	return &TelegramChannel{
		secrets: secrets,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Deliver sends a notification via Telegram.
func (t *TelegramChannel) Deliver(ctx context.Context, channel *NotificationChannel, event WebhookEvent) (rerr error) {
	var config map[string]string
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("parsing channel config: %w", err)
	}

	botToken, ok := config["bot_token"]
	if !ok || botToken == "" {
		return fmt.Errorf("bot_token missing from channel config")
	}

	// Decrypt bot token if it's a secret reference.
	if secretutil.IsSecretRef(botToken) {
		decrypted, err := t.secrets.RetrieveSecret(ctx, botToken)
		if err != nil {
			return fmt.Errorf("decrypting telegram bot_token: %w", err)
		}
		botToken = decrypted
	}

	chatID, ok := config["chat_id"]
	if !ok || chatID == "" {
		return fmt.Errorf("chat_id missing from channel config")
	}

	text := formatTelegramMessage(event)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req) //nolint:bodyclose // closed via multierr.AppendInvoke below
	if err != nil {
		return fmt.Errorf("sending telegram notification: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(resp.Body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func formatTelegramMessage(event WebhookEvent) string {
	msg := fmt.Sprintf("[Synclet] Sync %s", event.Event)
	if event.ConnectionID != "" {
		msg += fmt.Sprintf("\nConnection: %s", event.ConnectionID)
	}
	if event.Error != "" {
		msg += fmt.Sprintf("\nError: %s", event.Error)
	}
	return msg
}
