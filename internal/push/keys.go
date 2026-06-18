package push

import (
	webpush "github.com/SherClockHolmes/webpush-go"
)

// GenerateVAPIDKeys generates a new VAPID key pair (private, public, error).
// Use this once to populate VAPID_PRIVATE_KEY / VAPID_PUBLIC_KEY env vars.
func GenerateVAPIDKeys() (private, public string, err error) {
	return webpush.GenerateVAPIDKeys()
}
