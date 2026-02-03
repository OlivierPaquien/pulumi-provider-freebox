package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nikolalohinski/free-go/client"
	"github.com/pulumi/pulumi-go-provider/infer"
)

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
