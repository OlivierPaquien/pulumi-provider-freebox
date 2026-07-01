package main

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVpnServerPayload(t *testing.T) {
	defaults := freeboxTypes.OpenVPNServerConfig{
		Enabled:    false,
		ServerPort: 1194,
		ServerIP:   "10.8.0.0",
		ServerMask: "255.255.255.0",
		PushDHCP:   true,
		CA:         "existing-ca",
	}

	enabled := true
	port := int64(443)
	push := false
	payload := vpnServerPayload(VpnServerArgs{
		Enabled:    &enabled,
		ServerPort: &port,
		ServerIP:   "10.9.0.0",
		PushDHCP:   &push,
	}, defaults)

	assert.True(t, payload.Enabled)
	assert.Equal(t, int64(443), payload.ServerPort)
	assert.Equal(t, "10.9.0.0", payload.ServerIP)
	assert.Equal(t, "255.255.255.0", payload.ServerMask)
	assert.False(t, payload.PushDHCP)
	assert.Equal(t, "existing-ca", payload.CA)
}

func TestVpnServerFromConfig(t *testing.T) {
	state := vpnServerFromConfig(freeboxTypes.OpenVPNServerConfig{
		Enabled:    true,
		ServerPort: 1194,
		ServerIP:   "10.8.0.0",
		ServerMask: "255.255.255.0",
		PushDHCP:   false,
		CA:         "ca-data",
	})

	require.NotNil(t, state.Enabled)
	assert.True(t, *state.Enabled)
	require.NotNil(t, state.ServerPort)
	assert.Equal(t, int64(1194), *state.ServerPort)
	assert.Equal(t, "10.8.0.0", state.ServerIP)
	require.NotNil(t, state.PushDHCP)
	assert.False(t, *state.PushDHCP)
	assert.Equal(t, "ca-data", state.CA)
}

func TestVpnUserCreatePayload(t *testing.T) {
	payload := freeboxTypes.VPNUserPayload{
		Login:       "alice",
		Password:    "secret",
		Description: "test user",
	}
	assert.Equal(t, "alice", payload.Login)
	assert.Equal(t, "secret", payload.Password)
	assert.Equal(t, "test user", payload.Description)
}
