package main

import (
	"testing"
	"time"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVirtualMachinePowerInvalidPowerState(t *testing.T) {
	_, err := VirtualMachinePower{}.Create(nil, infer.CreateRequest[VirtualMachinePowerArgs]{
		Inputs: VirtualMachinePowerArgs{
			VmId:       1,
			PowerState: "paused",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), freeboxTypes.RunningStatus)

	_, err = VirtualMachinePower{}.Update(nil, infer.UpdateRequest[VirtualMachinePowerArgs, VirtualMachinePowerState]{
		Inputs: VirtualMachinePowerArgs{
			VmId:       1,
			PowerState: "booting",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), freeboxTypes.StoppedStatus)
}

func TestPowerKillTimeoutFromArgs(t *testing.T) {
	args := VirtualMachinePowerArgs{
		KillTimeout: 90,
	}
	assert.Equal(t, 90*time.Second, powerKillTimeout(args))
}
