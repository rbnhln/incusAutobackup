package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Gotify struct {
	Client  *http.Client
	PingURL string
}
type GotifyNotifier struct {
	gty *Gotify
}

func NewGotifyNotifier(pingURL string) *GotifyNotifier {
	return &GotifyNotifier{gty: NewGotify(pingURL)}
}

func (n GotifyNotifier) Name() string { return "gotify" }

func (n GotifyNotifier) Start(ctx context.Context) error {
	return nil
}

func (n GotifyNotifier) Finish(ctx context.Context, ok bool) error {
	return n.gty.Finish(ctx, ok)
}

func NewGotify(pingURL string) *Gotify {
	return &Gotify{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		PingURL: strings.TrimSpace(pingURL),
	}
}

func (g *Gotify) Finish(ctx context.Context, ok bool) error {
	if g == nil || g.Client == nil {
		return nil
	}
	if g.PingURL == "" {
		return nil
	}

	if !ok {
		title := "IAB Backup failed"
		message := "Backup run finished with errors"
		priority := 5

		payload := struct {
			Title    string `json:"title"`
			Message  string `json:"message"`
			Priority int    `json:"priority"`
		}{
			Title:    title,
			Message:  message,
			Priority: priority,
		}

		msg, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("gotify: marshal payload error: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.PingURL, bytes.NewReader(msg))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := g.Client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("gotify: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
	}
	return nil
}
