package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

const (
	envOpenVPNServerName    = "FREEBOX_VPN_SERVER"
	vpnClientConfigTimeout  = 90 * time.Second
	vpnClientConfigInterval = 2 * time.Second
	vpnServerReadyTimeout   = 3 * time.Minute
)

type vpnServerStatusV4 struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Type  string `json:"type"`
}

type vpnServerConfigV4 struct {
	Enabled     bool              `json:"enabled"`
	Port        int64             `json:"port"`
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	IPStart     string            `json:"ip_start"`
	IPEnd       string            `json:"ip_end"`
	ConfOpenVPN *vpnOpenVPNConfV4 `json:"conf_openvpn,omitempty"`
}

type vpnOpenVPNConfV4 struct {
	Cipher string `json:"cipher"`
}

type vpnServerUpdateV4 struct {
	Enabled *bool  `json:"enabled,omitempty"`
	Port    *int64 `json:"port,omitempty"`
}

type freeboxAPIResponse struct {
	Success   bool            `json:"success"`
	ErrorCode string          `json:"error_code"`
	Message   string          `json:"msg"`
	Result    json.RawMessage `json:"result"`
}

type freeboxSessionHTTP struct {
	baseURL string
	token   string
	http    client.HTTPClient
}

type authHeaderCapture struct {
	inner client.HTTPClient
	token string
}

func (c *authHeaderCapture) Do(req *http.Request) (*http.Response, error) {
	if token := req.Header.Get(client.AuthHeader); token != "" {
		c.token = token
	}
	return c.inner.Do(req)
}

func openVPNServerName() string {
	if name := os.Getenv(envOpenVPNServerName); name != "" {
		return name
	}
	return "openvpn_routed"
}

func isVPNLegacyAPIUnavailable(err error) bool {
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == "invalid_request" && strings.Contains(apiErr.Message, "404")
}

func isVPNBusy(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.Code == "busy"
}

func openVPNConfigFromV4(v4 vpnServerConfigV4) freeboxTypes.OpenVPNServerConfig {
	return freeboxTypes.OpenVPNServerConfig{
		Enabled:    v4.Enabled,
		ServerPort: v4.Port,
		ServerIP:   v4.IPStart,
	}
}

func vpnServerUpdateFromLegacy(payload freeboxTypes.OpenVPNServerConfig) vpnServerUpdateV4 {
	return vpnServerUpdateV4{
		Enabled: &payload.Enabled,
		Port:    &payload.ServerPort,
	}
}

func newFreeboxSessionHTTP(ctx context.Context, cfg Config) (*freeboxSessionHTTP, error) {
	baseURL, err := apiBaseURL(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AppID == "" || cfg.Token == "" {
		return nil, fmt.Errorf("Freebox provider requires appId and token (or FREEBOX_APP_ID and FREEBOX_TOKEN)")
	}

	capture := &authHeaderCapture{inner: http.DefaultClient}
	freeboxClient, err := client.New(cfg.Endpoint, cfg.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("create freebox client: %w", err)
	}
	freeboxClient = freeboxClient.WithAppID(cfg.AppID).WithPrivateToken(cfg.Token).WithHTTPClient(capture)
	if _, err := freeboxClient.Login(ctx); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	// Prime the session header used by authenticated API calls.
	_, _ = freeboxClient.ListVPNUsers(ctx)
	if capture.token == "" {
		return nil, fmt.Errorf("failed to capture freebox session token")
	}
	return &freeboxSessionHTTP{baseURL: baseURL, token: capture.token, http: capture.inner}, nil
}

func (h *freeboxSessionHTTP) getJSON(ctx context.Context, path string, target interface{}) error {
	body, contentType, err := h.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return decodeAPIResponse(body, contentType, target)
}

func (h *freeboxSessionHTTP) putJSON(ctx context.Context, path string, payload, target interface{}) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = strings.NewReader(string(encoded))
	}
	responseBody, contentType, err := h.do(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	return decodeAPIResponse(responseBody, contentType, target)
}

func (h *freeboxSessionHTTP) getRaw(ctx context.Context, path string) ([]byte, error) {
	body, contentType, err := h.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if strings.Contains(contentType, "application/json") {
		var resp freeboxAPIResponse
		if err := json.Unmarshal(body, &resp); err == nil && !resp.Success {
			return nil, &client.APIError{Code: resp.ErrorCode, Message: resp.Message}
		}
	}
	return body, nil
}

func (h *freeboxSessionHTTP) do(ctx context.Context, method, path string, body io.Reader) ([]byte, string, error) {
	requestURL := h.baseURL + "/" + strings.TrimPrefix(path, "/")
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(client.AuthHeader, h.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, "", fmt.Errorf("server error %d: %s", resp.StatusCode, string(responseBody))
	}
	return responseBody, resp.Header.Get("Content-Type"), nil
}

func decodeAPIResponse(body []byte, contentType string, target interface{}) error {
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unexpected content type %q", contentType)
	}
	var resp freeboxAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !resp.Success {
		return &client.APIError{Code: resp.ErrorCode, Message: resp.Message}
	}
	if target != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, target); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

func getOpenVPNServerConfigCompat(ctx context.Context, cli client.Client, cfg Config) (freeboxTypes.OpenVPNServerConfig, error) {
	config, err := cli.GetOpenVPNServerConfig(ctx)
	if err == nil {
		return config, nil
	}
	if !isVPNLegacyAPIUnavailable(err) {
		return config, err
	}
	return getOpenVPNServerConfigV4(ctx, cfg)
}

func getOpenVPNServerConfigV4(ctx context.Context, cfg Config) (freeboxTypes.OpenVPNServerConfig, error) {
	h, err := newFreeboxSessionHTTP(ctx, cfg)
	if err != nil {
		return freeboxTypes.OpenVPNServerConfig{}, err
	}
	var v4 vpnServerConfigV4
	path := fmt.Sprintf("vpn/%s/config/", openVPNServerName())
	if err := h.getJSON(ctx, path, &v4); err != nil {
		return freeboxTypes.OpenVPNServerConfig{}, fmt.Errorf("read OpenVPN server config (v4): %w", err)
	}
	return openVPNConfigFromV4(v4), nil
}

func updateOpenVPNServerConfigCompat(ctx context.Context, cli client.Client, cfg Config, payload freeboxTypes.OpenVPNServerConfig) (freeboxTypes.OpenVPNServerConfig, error) {
	updated, err := cli.UpdateOpenVPNServerConfig(ctx, payload)
	if err == nil {
		return updated, nil
	}
	if !isVPNLegacyAPIUnavailable(err) {
		return updated, err
	}
	return updateOpenVPNServerConfigV4(ctx, cfg, payload)
}

func updateOpenVPNServerConfigV4(ctx context.Context, cfg Config, payload freeboxTypes.OpenVPNServerConfig) (freeboxTypes.OpenVPNServerConfig, error) {
	h, err := newFreeboxSessionHTTP(ctx, cfg)
	if err != nil {
		return freeboxTypes.OpenVPNServerConfig{}, err
	}
	var v4 vpnServerConfigV4
	path := fmt.Sprintf("vpn/%s/config/", openVPNServerName())
	if err := h.putJSON(ctx, path, vpnServerUpdateFromLegacy(payload), &v4); err != nil {
		return freeboxTypes.OpenVPNServerConfig{}, fmt.Errorf("update OpenVPN server config (v4): %w", err)
	}
	return openVPNConfigFromV4(v4), nil
}

func usesV4VPNAPI(ctx context.Context, cli client.Client) bool {
	_, err := cli.GetOpenVPNServerConfig(ctx)
	return isVPNLegacyAPIUnavailable(err)
}

func ensureOpenVPNServerReadyV4(ctx context.Context, h *freeboxSessionHTTP) error {
	var v4 vpnServerConfigV4
	path := fmt.Sprintf("vpn/%s/config/", openVPNServerName())
	if err := h.getJSON(ctx, path, &v4); err != nil {
		return fmt.Errorf("read OpenVPN server config (v4): %w", err)
	}
	if !v4.Enabled {
		enabled := true
		update := vpnServerUpdateV4{Enabled: &enabled, Port: &v4.Port}
		if err := h.putJSON(ctx, path, update, &v4); err != nil {
			return fmt.Errorf("enable OpenVPN server: %w", err)
		}
	}
	return waitForOpenVPNServerStarted(ctx, h)
}

func waitForOpenVPNServerStarted(ctx context.Context, h *freeboxSessionHTTP) error {
	deadline := time.Now().Add(vpnServerReadyTimeout)
	name := openVPNServerName()
	var lastState string
	for {
		var servers []vpnServerStatusV4
		if err := h.getJSON(ctx, "vpn/", &servers); err != nil {
			return fmt.Errorf("list VPN servers: %w", err)
		}
		for _, server := range servers {
			if server.Name != name {
				continue
			}
			lastState = server.State
			switch server.State {
			case "started":
				return nil
			case "error":
				return fmt.Errorf("openvpn server %q is in error state", name)
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("openvpn server %q not started after %s (last state=%q)", name, vpnServerReadyTimeout, lastState)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(vpnClientConfigInterval):
		}
	}
}

func getVPNUserClientConfigCompat(ctx context.Context, cli client.Client, cfg Config, login string) (string, error) {
	if !usesV4VPNAPI(ctx, cli) {
		return pollVPNClientConfig(ctx, func() (string, error) {
			config, err := cli.GetVPNUserClientConfig(ctx, login)
			if err == nil {
				return config, nil
			}
			if errors.Is(err, client.ErrVPNUserNotFound) {
				return "", err
			}
			return "", err
		})
	}

	h, err := newFreeboxSessionHTTP(ctx, cfg)
	if err != nil {
		return "", err
	}
	if err := ensureOpenVPNServerReadyV4(ctx, h); err != nil {
		return "", err
	}
	return pollVPNClientConfig(ctx, func() (string, error) {
		path := fmt.Sprintf("vpn/download_config/%s/%s", openVPNServerName(), login)
		body, err := h.getRaw(ctx, path)
		if err != nil {
			return "", err
		}
		return string(body), nil
	})
}

func updateVPNUserCompat(ctx context.Context, cli client.Client, cfg Config, login, password string) (freeboxTypes.VPNUser, error) {
	if !usesV4VPNAPI(ctx, cli) {
		return cli.UpdateVPNUser(ctx, login, freeboxTypes.VPNUserPayload{
			Login:    login,
			Password: password,
		})
	}
	// Modern Freebox VPN API rejects PUT password updates (inval); recreate the user instead.
	return recreateVPNUser(ctx, cli, login, password)
}

func recreateVPNUser(ctx context.Context, cli client.Client, login, password string) (freeboxTypes.VPNUser, error) {
	if err := cli.DeleteVPNUser(ctx, login); err != nil && !errors.Is(err, client.ErrVPNUserNotFound) {
		return freeboxTypes.VPNUser{}, fmt.Errorf("delete VPN user before recreate: %w", err)
	}
	user, err := cli.CreateVPNUser(ctx, freeboxTypes.VPNUserPayload{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return freeboxTypes.VPNUser{}, fmt.Errorf("recreate VPN user: %w", err)
	}
	return user, nil
}

func pollVPNClientConfig(ctx context.Context, fetch func() (string, error)) (string, error) {
	deadline := time.Now().Add(vpnClientConfigTimeout)
	var lastErr error
	for {
		config, err := fetch()
		if err == nil {
			return config, nil
		}
		if !isVPNBusy(err) {
			return "", err
		}
		lastErr = err
		if time.Now().After(deadline) {
			return "", fmt.Errorf("vpn client config not ready after %s: %w", vpnClientConfigTimeout, lastErr)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(vpnClientConfigInterval):
		}
	}
}
