package notifier

import (
	"context"
	"fmt"
	"net/http"
)

type httpNotifier struct {
	method string
	url    string

	httpClient *http.Client
}

func NewHTTP(url, method string) Notifier {
	return &httpNotifier{
		url:        url,
		method:     method,
		httpClient: http.DefaultClient,
	}
}

func (n *httpNotifier) Notify(ctx context.Context) error {
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
