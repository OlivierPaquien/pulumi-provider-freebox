package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nikolalohinski/free-go/client"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// retryTransport retries on 502/503/504 (transient gateway errors).
type retryTransport struct {
	rt         http.RoundTripper
	maxRetries int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		resp, err = t.rt.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 502 && resp.StatusCode != 503 && resp.StatusCode != 504 {
			return resp, nil
		}
		// Only retry GET/DELETE (no body); POST/PUT body is already consumed
		if req.Body != nil {
			return resp, nil
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if attempt < t.maxRetries {
			backoff := time.Duration(attempt+1) * 2 * time.Second
			time.Sleep(backoff)
			retryReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), nil)
			if err != nil {
				return resp, nil
			}
			retryReq.Header = req.Header.Clone()
			req = retryReq
		} else {
			return resp, nil
		}
	}
	return resp, nil
}

// debugTransport logs HTTP requests and responses when FREEBOX_DEBUG_HTTP=1.
type debugTransport struct {
	rt http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	freeboxLog("[freebox http] %s %s\n", req.Method, req.URL.String())
	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		freeboxLog("[freebox http] error: %v\n", err)
		return nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	freeboxLog("[freebox http] %d body=%s\n", resp.StatusCode, truncate(string(body), 500))
	return resp, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

const (
	defaultEndpoint = "http://mafreebox.freebox.fr"
	defaultVersion  = "latest"
	envEndpoint     = "FREEBOX_ENDPOINT"
	envVersion      = "FREEBOX_VERSION"
	envAppID        = "FREEBOX_APP_ID"
	envToken        = "FREEBOX_TOKEN"
)

// getFreeboxClient returns a configured Freebox API client from provider config or env.
func getFreeboxClient(ctx context.Context) (client.Client, error) {
	cfg := infer.GetConfig[Config](ctx)

	endpoint := defaultEndpoint
	if v := os.Getenv(envEndpoint); v != "" {
		endpoint = v
	}
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	version := defaultVersion
	if v := os.Getenv(envVersion); v != "" {
		version = v
	}
	if cfg.APIVersion != "" {
		version = cfg.APIVersion
	}

	appID := os.Getenv(envAppID)
	if cfg.AppID != "" {
		appID = cfg.AppID
	}
	token := os.Getenv(envToken)
	if cfg.Token != "" {
		token = cfg.Token
	}

	if appID == "" || token == "" {
		return nil, fmt.Errorf("Freebox provider requires appId and token (or FREEBOX_APP_ID and FREEBOX_TOKEN)")
	}

	c, err := client.New(endpoint, version)
	if err != nil {
		return nil, fmt.Errorf("create freebox client: %w", err)
	}
	c = c.WithAppID(appID).WithPrivateToken(token)
	return c, nil
}
