package notifyservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

type ownedSecret struct {
	ownerType string
	ownerID   uuid.UUID
	plaintext string
}

// ownerCheckedSecrets resolves a reference only for the owner it was stored under.
type ownerCheckedSecrets struct {
	SecretsProvider
	secrets map[string]ownedSecret
}

func newOwnerCheckedSecrets() *ownerCheckedSecrets {
	return &ownerCheckedSecrets{secrets: map[string]ownedSecret{}}
}

func (f *ownerCheckedSecrets) store(ownerType string, ownerID uuid.UUID, plaintext string) string {
	ref := secretutil.MakeSecretRef(uuid.New())
	f.secrets[ref] = ownedSecret{ownerType: ownerType, ownerID: ownerID, plaintext: plaintext}

	return ref
}

func (f *ownerCheckedSecrets) RetrieveSecret(_ context.Context, ownerType string, ownerID uuid.UUID, ref string) (string, error) {
	secret, ok := f.secrets[ref]
	if !ok || secret.ownerType != ownerType || secret.ownerID != ownerID {
		return "", NotFoundError("Secret")
	}

	return secret.plaintext, nil
}

// routedClient sends every request to server, whatever host the URL names.
func routedClient(server *httptest.Server) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()      //nolint:forcetypeassert // DefaultTransport is always *http.Transport
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // test server certificate
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}

	return &http.Client{Timeout: 5 * time.Second, Transport: transport}
}

type ownerWebhookStorage struct {
	*testStorage
	webhook *Webhook
}

func (s *ownerWebhookStorage) Webhooks() WebhooksStorage { return &ownerWebhooks{webhook: s.webhook} }

type ownerWebhooks struct {
	WebhooksStorage
	webhook *Webhook
}

func (s *ownerWebhooks) First(context.Context, *WebhookFilter, ...optionutil.Option[dbutil.SelectOptions]) (*Webhook, error) {
	cp := *s.webhook

	return &cp, nil
}

func (s *ownerWebhooks) Find(context.Context, *WebhookFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*Webhook, error) {
	cp := *s.webhook

	return []*Webhook{&cp}, nil
}

// signedWebhook returns a webhook signed with a secret held by the webhook and
// a server that records the signature it receives.
func signedWebhook(t *testing.T, secrets *ownerCheckedSecrets) (*Webhook, *httptest.Server, *[]string) {
	t.Helper()

	signatures := &[]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mac := hmac.New(sha256.New, []byte("signing-key"))
		mac.Write(body)

		if r.Header.Get("X-Synclet-Signature") == hex.EncodeToString(mac.Sum(nil)) {
			*signatures = append(*signatures, "valid")
		}

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	webhook := &Webhook{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		URL:         "http://1.1.1.1/hook",
		Events:      `["sync.failed"]`,
		Enabled:     true,
	}
	webhook.Secret = secrets.store("webhook", webhook.ID, "signing-key")

	return webhook, server, signatures
}

func TestDeliverWebhook_SignsWithTheWebhooksSecret(t *testing.T) {
	secrets := newOwnerCheckedSecrets()
	webhook, server, signatures := signedWebhook(t, secrets)

	uc := NewDeliverWebhookWithClient(&ownerWebhookStorage{webhook: webhook}, secrets, Config{WebhookMaxRetries: 1}, routedClient(server))
	require.NoError(t, uc.Execute(context.Background(), DeliverWebhookParams{
		WorkspaceID: webhook.WorkspaceID,
		Event:       WebhookEvent{Event: "sync.failed", Timestamp: time.Now()},
	}))

	assert.Equal(t, []string{"valid"}, *signatures)
}

func TestTestWebhook_SignsWithTheWebhooksSecret(t *testing.T) {
	secrets := newOwnerCheckedSecrets()
	webhook, server, signatures := signedWebhook(t, secrets)

	uc := NewTestWebhook(&ownerWebhookStorage{webhook: webhook}, secrets, Config{WebhookHTTPTimeout: 5 * time.Second}, &capturingAuditRecorder{})
	uc.httpClient = routedClient(server)

	result, err := uc.Execute(context.Background(), TestWebhookParams{ID: webhook.ID, WorkspaceID: webhook.WorkspaceID})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, []string{"valid"}, *signatures)
}

func TestSlackChannel_Deliver_DecryptsTheChannelsWebhookURL(t *testing.T) {
	var hits int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secrets := newOwnerCheckedSecrets()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: uuid.New(), ChannelType: ChannelTypeSlack, Enabled: true}
	channel.Config = mustJSON(map[string]string{"webhook_url": secrets.store("channel", channel.ID, server.URL)})

	err := NewSlackChannelWithClient(secrets, server.Client()).Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"})
	require.NoError(t, err)
	assert.Equal(t, 1, hits)
}

func TestTelegramChannel_Deliver_DecryptsTheChannelsBotToken(t *testing.T) {
	var paths []string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secrets := newOwnerCheckedSecrets()
	channel := &NotificationChannel{ID: uuid.New(), WorkspaceID: uuid.New(), ChannelType: ChannelTypeTelegram, Enabled: true}
	channel.Config = mustJSON(map[string]string{
		"bot_token": secrets.store("channel", channel.ID, "123:token"),
		"chat_id":   "42",
	})

	telegram := NewTelegramChannel(secrets, Config{WebhookHTTPTimeout: 5 * time.Second})
	telegram.httpClient = routedClient(server)

	require.NoError(t, telegram.Deliver(context.Background(), channel, WebhookEvent{Event: "sync.failed"}))
	assert.Equal(t, []string{"/bot123:token/sendMessage"}, paths)
}
