package notifications

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Healthchecks struct {
	Client  *http.Client
	PingURL string
}

type HealthchecksNotifier struct {
	hc *Healthchecks
}

func NewHealthchecksNotifier(pingURL string) *HealthchecksNotifier {
	return &HealthchecksNotifier{hc: NewHealthchecks(pingURL)}
}

func (n *HealthchecksNotifier) Name() string { return "healthchecks" }

func (n *HealthchecksNotifier) Start(ctx context.Context) error {
	return n.hc.Start(ctx)
}

func (n *HealthchecksNotifier) Finish(ctx context.Context, ok bool) error {
	status := 0
	if !ok {
		status = 1
	}
	return n.hc.Status(ctx, status)
}

func NewHealthchecks(pingURL string) *Healthchecks {
	return &Healthchecks{
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		PingURL: strings.TrimRight(strings.TrimSpace(pingURL), "/"),
	}
}

func (h *Healthchecks) Start(ctx context.Context) error {
	return h.ping(ctx, "/start")
}

func (h *Healthchecks) Status(ctx context.Context, status int) error {
	if status != 0 && status != 1 {
		return fmt.Errorf("healthchecks: invalid status %d (expected 0 or 1)", status)
	}
	return h.ping(ctx, fmt.Sprintf("/%d", status))
}

func (h *Healthchecks) ping(ctx context.Context, suffix string) error {
	if h == nil || h.Client == nil {
		return nil
	}
	if h.PingURL == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, h.PingURL+suffix, nil)
	if err != nil {
		return err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("healthchecks: unexpected status %s", resp.Status)
	}

	return nil
}
