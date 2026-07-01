package main

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeMAC(t *testing.T) {
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", normalizeMAC("aa:bb:cc:dd:ee:ff"))
}

func TestDhcpLeaseHostname(t *testing.T) {
	lease := freeboxTypes.DHCPStaticLeaseInfo{
		Mac:      "2E:7E:55:60:5A:66",
		Hostname: "2E:7E:55:60:5A:66",
	}
	assert.Equal(t, "vm-ubuntu", dhcpLeaseHostname(lease, "vm-ubuntu"))
	lease.Hostname = "real-host"
	assert.Equal(t, "real-host", dhcpLeaseHostname(lease, "vm-ubuntu"))
}

func TestIsValidIPv4(t *testing.T) {
	assert.True(t, isValidIPv4("192.168.1.10"))
	assert.False(t, isValidIPv4("not-an-ip"))
	assert.False(t, isValidIPv4("::1"))
}

func TestIsValidMAC(t *testing.T) {
	assert.True(t, isValidMAC("aa:bb:cc:dd:ee:ff"))
	assert.True(t, isValidMAC("AA-BB-CC-DD-EE-FF"))
	assert.False(t, isValidMAC("not-a-mac"))
	assert.False(t, isValidMAC(""))
}

func TestValidateDHCPStaticLeaseArgs(t *testing.T) {
	require.NoError(t, validateDHCPStaticLeaseArgs("AA:BB:CC:DD:EE:FF", "192.168.1.10"))

	err := validateDHCPStaticLeaseArgs("bad-mac", "192.168.1.10")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mac")

	err = validateDHCPStaticLeaseArgs("AA:BB:CC:DD:EE:FF", "not-an-ip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ip")
}

func TestNormalizePowerState(t *testing.T) {
	assert.Equal(t, "running", normalizePowerState("starting"))
	assert.Equal(t, "stopped", normalizePowerState("stopping"))
}
