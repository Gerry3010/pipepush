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

// pushPayload is the JSON sent to the browser service worker.
// Only the encrypted blob and a run id travel through the push service —
// the actual run details are inside encryptedPayload (E2E).
type pushPayload struct {
	Type             string `json:"type"`
	RunID            string `json:"runId"`
	Status           string `json:"status"`
	EncryptedPayload string `json:"encryptedPayload"`
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
		Type:             "run_update",
		RunID:            run.ID,
		Status:           run.Status,
		EncryptedPayload: run.EncryptedPayload,
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
