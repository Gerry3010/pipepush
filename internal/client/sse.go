package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Gerry3010/pipepush/internal/models"
)

// StreamEvents connects to the SSE endpoint and invokes onEvent for each run update.
// It blocks until the context is cancelled or the connection fails.
func (c *Client) StreamEvents(ctx context.Context, onEvent func(models.SSEEvent)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/events", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("Accept", "text/event-stream")

	// No timeout for streaming connections
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var dataLines []string
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		switch {
		case line == "":
			// dispatch accumulated event
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				var event models.SSEEvent
				if json.Unmarshal([]byte(data), &event) == nil {
					onEvent(event)
				}
				dataLines = dataLines[:0]
			}
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return scanner.Err()
}
