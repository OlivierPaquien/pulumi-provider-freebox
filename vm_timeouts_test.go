package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVirtualMachineTimeoutsResolved(t *testing.T) {
	def := defaultVirtualMachineTimeouts()
	assert.Equal(t, 5*time.Minute, def.Create)
	assert.Equal(t, 30*time.Second, def.Kill)

	var nilTimeouts *VirtualMachineTimeouts
	resolved := nilTimeouts.resolved()
	assert.Equal(t, def, resolved)

	custom := &VirtualMachineTimeouts{Kill: 10 * time.Second}
	resolved = custom.resolved()
	assert.Equal(t, 10*time.Second, resolved.Kill)
	assert.Equal(t, 5*time.Minute, resolved.Create)
}

func TestVirtualMachinePowerKillTimeout(t *testing.T) {
	var nilTimeouts *VirtualMachinePowerTimeouts
	assert.Equal(t, 30*time.Second, nilTimeouts.killTimeout(0))
	assert.Equal(t, 45*time.Second, nilTimeouts.killTimeout(45))

	custom := &VirtualMachinePowerTimeouts{Kill: 2 * time.Minute}
	assert.Equal(t, 2*time.Minute, custom.killTimeout(45))
}
