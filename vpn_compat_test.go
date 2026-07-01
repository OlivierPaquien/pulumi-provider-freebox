package main

import (
	"testing"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsVPNLegacyAPIUnavailable(t *testing.T) {
	assert.True(t, isVPNLegacyAPIUnavailable(&client.APIError{
		Code:    "invalid_request",
		Message: "Requête invalide (404)",
	}))
	assert.False(t, isVPNLegacyAPIUnavailable(&client.APIError{
		Code:    "noent",
		Message: "not found",
	}))
	assert.False(t, isVPNLegacyAPIUnavailable(assert.AnError))
}

func TestIsVPNBusy(t *testing.T) {
	assert.True(t, isVPNBusy(&client.APIError{Code: "busy", Message: "not ready"}))
	assert.False(t, isVPNBusy(&client.APIError{Code: "noent"}))
}

func TestOpenVPNConfigFromV4(t *testing.T) {
	cfg := openVPNConfigFromV4(vpnServerConfigV4{
		Enabled: true,
		Port:    1194,
		IPStart: "192.168.27.65",
		IPEnd:   "192.168.27.95",
		ID:      "openvpn_routed",
		Type:    "openvpn",
	})
	assert.True(t, cfg.Enabled)
	assert.Equal(t, int64(1194), cfg.ServerPort)
	assert.Equal(t, "192.168.27.65", cfg.ServerIP)
}

func TestVpnServerUpdateFromLegacy(t *testing.T) {
	payload := freeboxTypes.OpenVPNServerConfig{Enabled: true, ServerPort: 1194}
	upd := vpnServerUpdateFromLegacy(payload)
	require.NotNil(t, upd.Enabled)
	require.NotNil(t, upd.Port)
	assert.True(t, *upd.Enabled)
	assert.Equal(t, int64(1194), *upd.Port)
}
