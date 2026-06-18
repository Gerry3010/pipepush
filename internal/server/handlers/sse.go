package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Gerry3010/pipepush/internal/models"
	"github.com/Gerry3010/pipepush/internal/server/middleware"
)

// SSEHub manages per-user Server-Sent Event subscribers for realtime run updates.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan models.SSEEvent]struct{} // userID -> set of channels
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[string]map[chan models.SSEEvent]struct{})}
}

func (h *SSEHub) subscribe(userID string) chan models.SSEEvent {
	ch := make(chan models.SSEEvent, 8)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[chan models.SSEEvent]struct{})
	}
	h.clients[userID][ch] = struct{}{}
	return ch
}

func (h *SSEHub) unsubscribe(userID string, ch chan models.SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.clients[userID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.clients, userID)
		}
	}
	close(ch)
}

// Broadcast sends an event to all connected clients of a user (non-blocking).
func (h *SSEHub) Broadcast(userID string, event models.SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[userID] {
		select {
		case ch <- event:
		default: // drop if client is slow; it will catch up via REST history
		}
	}
}

// Handler streams run updates to the authenticated user over SSE.
func (h *SSEHub) Handler(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserIDFromContext(r.Context())

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.subscribe(uid)
	defer h.unsubscribe(uid, ch)

	// Initial comment to open the stream
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-ch:
			data, _ := json.Marshal(event)
			_, _ = w.Write([]byte("event: " + event.Type + "\ndata: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
