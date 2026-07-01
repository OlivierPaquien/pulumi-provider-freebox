package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

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
	cfg := providerConfig(ctx)
	if cfg.AppID == "" || cfg.Token == "" {
		return nil, fmt.Errorf("Freebox provider requires appId and token (or FREEBOX_APP_ID and FREEBOX_TOKEN)")
	}

	freeboxLog("[freebox] client endpoint=%q apiVersion=%q appId=%q\n", cfg.Endpoint, cfg.APIVersion, cfg.AppID)

	c, err := client.New(cfg.Endpoint, cfg.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("create freebox client: %w", err)
	}
	c = c.WithAppID(cfg.AppID).WithPrivateToken(cfg.Token)
	return c, nil
}

// providerConfig merges Pulumi provider config with environment variables.
func providerConfig(ctx context.Context) Config {
	return mergeConfigWithEnv(inferConfigIfPresent(ctx))
}

func inferConfigIfPresent(ctx context.Context) (cfg Config) {
	defer func() {
		if recover() != nil {
			cfg = Config{}
		}
	}()
	cfg = infer.GetConfig[Config](ctx)
	return cfg
}

func mergeConfigWithEnv(cfg Config) Config {
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

	return Config{
		Endpoint:   endpoint,
		APIVersion: version,
		AppID:      appID,
		Token:      token,
	}
}

func apiBaseURL(cfg Config) (string, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	version := cfg.APIVersion
	if version == "" {
		version = defaultVersion
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	base, err := url.Parse(fmt.Sprintf("%s/api/%s", endpoint, version))
	if err != nil {
		return "", fmt.Errorf("build api base url: %w", err)
	}
	return strings.TrimRight(base.String(), "/"), nil
}
