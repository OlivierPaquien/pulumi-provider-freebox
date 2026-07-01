package main

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortForwardingPayload(t *testing.T) {
	end := int64(8081)
	target := int64(80)
	args := PortForwardingArgs{
		Enabled:        true,
		IPProtocol:     "tcp",
		PortRangeStart: 8080,
		PortRangeEnd:   &end,
		TargetPort:     &target,
		TargetIP:       "192.168.1.10",
		SourceIP:       "1.2.3.4",
		Comment:        "test",
	}

	payload := portForwardingPayload(args)
	require.NotNil(t, payload.Enabled)
	assert.True(t, *payload.Enabled)
	assert.Equal(t, "tcp", string(payload.IPProtocol))
	assert.Equal(t, int64(8080), payload.WanPortStart)
	assert.Equal(t, int64(8081), payload.WanPortEnd)
	assert.Equal(t, int64(80), payload.LanPort)
	assert.Equal(t, "192.168.1.10", payload.LanIP)
	assert.Equal(t, "1.2.3.4", payload.SourceIP)
	assert.Equal(t, "test", payload.Comment)
}

func TestPortForwardingPayloadDefaults(t *testing.T) {
	args := PortForwardingArgs{
		Enabled:        false,
		IPProtocol:     "udp",
		PortRangeStart: 53,
		TargetIP:       "192.168.1.1",
	}

	payload := portForwardingPayload(args)
	assert.Equal(t, int64(53), payload.WanPortEnd)
	assert.Equal(t, int64(53), payload.LanPort)
	assert.Equal(t, "0.0.0.0", payload.SourceIP)
}

func TestPortForwardingArgsFromRuleRoundTrip(t *testing.T) {
	enabled := true
	end := int64(443)
	target := int64(443)
	args := PortForwardingArgs{
		Enabled:        true,
		IPProtocol:     "tcp",
		PortRangeStart: 443,
		PortRangeEnd:   &end,
		TargetPort:     &target,
		SourceIP:       "0.0.0.0",
		TargetIP:       "192.168.1.5",
		Comment:        "https",
	}
	payload := portForwardingPayload(args)

	rule := freeboxTypes.PortForwardingRule{
		PortForwardingRulePayload: payload,
		ID:                        42,
		Hostname:                  "host.local",
	}
	rule.Enabled = &enabled

	roundTrip := portForwardingArgsFromRule(rule)
	assert.Equal(t, args.Enabled, roundTrip.Enabled)
	assert.Equal(t, args.IPProtocol, roundTrip.IPProtocol)
	assert.Equal(t, args.PortRangeStart, roundTrip.PortRangeStart)
	require.NotNil(t, roundTrip.PortRangeEnd)
	assert.Equal(t, *args.PortRangeEnd, *roundTrip.PortRangeEnd)
	require.NotNil(t, roundTrip.TargetPort)
	assert.Equal(t, *args.TargetPort, *roundTrip.TargetPort)
	assert.Equal(t, args.SourceIP, roundTrip.SourceIP)
	assert.Equal(t, args.TargetIP, roundTrip.TargetIP)
	assert.Equal(t, args.Comment, roundTrip.Comment)

	state := portForwardingStateFromRule(args, rule)
	assert.Equal(t, int64(42), state.ID)
	assert.Equal(t, "host.local", state.Hostname)
}
