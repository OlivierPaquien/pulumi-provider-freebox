package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const (
	mockAPIVersion   = "v0"
	mockAppID        = "test"
	mockPrivateToken = "xXXyyX9999wwwwwwwwxxx99999XXYYYYYYWWW000000000999999XXXXX9999Yx"
	mockSessionToken = "EfETzVibY7K5vZVsq+MjtD6pDJoAaYQiqyXwS5kFvooTczPMk7Tz+6//aTe9zZNy"
)

// sequentialMockServer dispatches HTTP requests to handlers in registration order.
type sequentialMockServer struct {
	t        *testing.T
	handlers []http.HandlerFunc
	index    int
}

func (s *sequentialMockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.index >= len(s.handlers) {
		s.t.Fatalf("unexpected request #%d: %s %s", s.index, r.Method, r.URL.Path)
	}
	h := s.handlers[s.index]
	s.index++
	h(w, r)
}

func (s *sequentialMockServer) append(h http.HandlerFunc) {
	s.handlers = append(s.handlers, h)
}

func newSequentialMockServer(t *testing.T) (*httptest.Server, *sequentialMockServer) {
	seq := &sequentialMockServer{t: t}
	seq.handlers = mockLoginHandlers()
	srv := httptest.NewServer(seq)
	return srv, seq
}

func mockTestConfig(endpoint string) Config {
	return Config{
		Endpoint:   endpoint,
		APIVersion: mockAPIVersion,
		AppID:      mockAppID,
		Token:      mockPrivateToken,
	}
}

func mockLoginHandlers() []http.HandlerFunc {
	return []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/login") {
				http.Error(w, "expected GET login", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, `{
				"success": true,
				"result": {
					"logged_in": false,
					"challenge": "9Va31tSgQWM853j0kSCtBUyzYNhPN7IY"
				}
			}`)
		},
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/login/session") {
				http.Error(w, "expected POST login/session", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, `{
				"success": true,
				"result": {
					"session_token": "`+mockSessionToken+`",
					"challenge": "9Va31tSgQWM853j0kSCtBUyzYNhPN7IY",
					"permissions": {}
				}
			}`)
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func mockPortForwardingRuleJSON(id int64, wanPort int64, lanIP string) string {
	return `{
		"success": true,
		"result": {
			"enabled": true,
			"comment": "pulumi-test",
			"id": ` + formatInt(id) + `,
			"valid": true,
			"src_ip": "0.0.0.0",
			"hostname": "host.local",
			"lan_port": ` + formatInt(wanPort) + `,
			"wan_port_end": ` + formatInt(wanPort) + `,
			"wan_port_start": ` + formatInt(wanPort) + `,
			"lan_ip": "` + lanIP + `",
			"ip_proto": "tcp"
		}
	}`
}

func mockVirtualMachineJSON(id int64, name, status string) string {
	return `{
		"success": true,
		"result": {
			"id": ` + formatInt(id) + `,
			"name": "` + name + `",
			"status": "` + status + `",
			"mac": "f6:69:9c:d9:4f:3d",
			"disk_path": "L0ZyZWVib3gvZGlzay1wYXRo",
			"disk_type": "qcow2",
			"memory": 256,
			"vcpus": 1,
			"enable_cloudinit": false,
			"bind_usb_ports": ""
		}
	}`
}

func formatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func mockOpenVPNServerJSON(enabled bool, port int64) string {
	enabledStr := "false"
	if enabled {
		enabledStr = "true"
	}
	return `{
		"success": true,
		"result": {
			"enabled": ` + enabledStr + `,
			"server_port": ` + formatInt(port) + `,
			"server_ip": "10.8.0.0",
			"server_mask": "255.255.255.0",
			"push_default_gw": false,
			"push_dhcp": true,
			"ca": "-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----\n"
		}
	}`
}

func mockVPNUserJSON(login, description string) string {
	return `{
		"success": true,
		"result": {
			"login": "` + login + `",
			"password": "",
			"description": "` + description + `"
		}
	}`
}

func mockVPNUserNotFoundJSON() string {
	return `{
		"success": false,
		"error_code": "noent",
		"msg": "User not found"
	}`
}

func mockVPNClientConfigJSON(content string) string {
	// result is a JSON string value
	escaped := strings.ReplaceAll(content, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	return `{
		"success": true,
		"result": "` + escaped + `"
	}`
}

func mockAPIError404JSON() string {
	return `{
		"success": false,
		"error_code": "invalid_request",
		"msg": "Requête invalide (404)"
	}`
}

func mockOpenVPNServerV4JSON(enabled bool, port int64) string {
	enabledStr := "false"
	if enabled {
		enabledStr = "true"
	}
	return `{
		"success": true,
		"result": {
			"enabled": ` + enabledStr + `,
			"port": ` + formatInt(port) + `,
			"id": "openvpn_routed",
			"type": "openvpn",
			"ip_start": "192.168.27.65",
			"ip_end": "192.168.27.95",
			"conf_openvpn": {"cipher": "aes128"}
		}
	}`
}

func mockVPNServerListV4JSON(serverName, state string) string {
	return `{
		"success": true,
		"result": [
			{
				"name": "` + serverName + `",
				"state": "` + state + `",
				"type": "openvpn",
				"connection_count": 0,
				"auth_connection_count": 0
			}
		]
	}`
}
