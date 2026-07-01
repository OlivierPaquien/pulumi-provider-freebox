package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateVPNUserPassword(t *testing.T) {
	require.NoError(t, validateVPNUserPassword("12345678"))
	require.NoError(t, validateVPNUserPassword(strings.Repeat("a", 32)))
	require.Error(t, validateVPNUserPassword("short"))
	require.Error(t, validateVPNUserPassword(strings.Repeat("a", 33)))
}

func TestVpnUserDescriptionFromState(t *testing.T) {
	state := VpnUserState{VpnUserArgs: VpnUserArgs{Description: "from-state"}}
	inputs := VpnUserArgs{Description: "from-inputs"}
	assert.Equal(t, "from-inputs", vpnUserDescriptionFromState(state, inputs, "from-api"))
	assert.Equal(t, "from-state", vpnUserDescriptionFromState(state, VpnUserArgs{}, "from-api"))
	assert.Equal(t, "from-api", vpnUserDescriptionFromState(VpnUserState{}, VpnUserArgs{}, "from-api"))
}
