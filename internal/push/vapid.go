package push

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/Gerry3010/pipepush/internal/db"
	"github.com/Gerry3010/pipepush/internal/models"
)

// Dispatcher sends Web Push notifications to a user's subscribed devices.
type Dispatcher struct {
	db           *db.DB
	vapidPublic  string
	vapidPrivate string
	vapidEmail   string
}

func NewDispatcher(database *db.DB, vapidPublic, vapidPrivate, vapidEmail string) *Dispatcher {
	return &Dispatcher{
		db:           database,
		vapidPublic:  vapidPublic,
		vapidPrivate: vapidPrivate,
		vapidEmail:   vapidEmail,
	}
}

// Enabled reports whether VAPID keys are configured.
func (d *Dispatcher) Enabled() bool {
	return d.vapidPublic != "" && d.vapidPrivate != ""
}

// pushPayload is the JSON sent to the browser service worker. It is deliberately
// tiny — only non-sensitive routing metadata travels through the push service.
// The run details (and logs) are NOT included: they can be large and would blow
// the ~4KB Web Push limit, and keeping them out means nothing sensitive passes
// through the push provider. Open pages refetch on the "run_update" signal, and
// the run-detail view fetches the full E2E payload via GET /runs/{id}.
type pushPayload struct {
	Type  string `json:"type"`
	RunID string `json:"runId"`
	// PipelineID is a non-sensitive UUID (not a name). The service worker uses it
	// to look up the locally-cached, client-decrypted pipeline/project name so the
	// notification can show it — the plaintext name never travels through the push
	// service, preserving E2E.
	PipelineID string `json:"pipelineId"`
	Status     string `json:"status"`
}

// SendRunNotification pushes an encrypted run update to all of the user's devices.
func (d *Dispatcher) SendRunNotification(ctx context.Context, userID string, run *models.Run) {
	if !d.Enabled() {
		return
	}

	subs, err := d.db.ListVAPIDSubscriptions(ctx, userID)
	if err != nil {
		slog.Error("push: listing subscriptions failed", "error", err)
		return
	}

	payload, _ := json.Marshal(pushPayload{
		Type:       "run_update",
		RunID:      run.ID,
		PipelineID: run.PipelineID,
		Status:     run.Status,
	})

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256DHKey,
				Auth:   sub.AuthKey,
			},
		}

		resp, err := webpush.SendNotificationWithContext(ctx, payload, s, &webpush.Options{
			Subscriber:      d.vapidEmail,
			VAPIDPublicKey:  d.vapidPublic,
			VAPIDPrivateKey: d.vapidPrivate,
			TTL:             86400,
		})
		if err != nil {
			slog.Warn("push: send failed", "endpoint", sub.Endpoint, "error", err)
			continue
		}
		// 404/410 mean the subscription is dead — clean it up.
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			_ = d.db.DeleteVAPIDSubscription(ctx, userID, sub.Endpoint)
		}
		resp.Body.Close()
	}
}
