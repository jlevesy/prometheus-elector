package notifier

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type httpNotifier struct {
	method string
	url    string

	timeout time.Duration

	httpClient *http.Client
}

func NewHTTP(url, method string, timeout time.Duration) Notifier {
	return &httpNotifier{
		url:        url,
		method:     method,
		timeout:    timeout,
		httpClient: http.DefaultClient,
	}
}

func (n *httpNotifier) Notify(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, n.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, n.method, n.url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
