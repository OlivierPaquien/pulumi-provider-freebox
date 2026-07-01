package main

import (
	"context"
	"testing"
	"time"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesiredVMStatus(t *testing.T) {
	assert.Equal(t, freeboxTypes.RunningStatus, desiredVMStatus(VirtualMachineArgs{}))
	assert.Equal(t, freeboxTypes.StoppedStatus, desiredVMStatus(VirtualMachineArgs{Status: freeboxTypes.StoppedStatus}))
}

func TestApplyVMDesiredStatusUnsupported(t *testing.T) {
	_, err := applyVMDesiredStatus(context.Background(), nil, 1, "paused", time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported VM status")
}

func TestEnsureVMStoppedForConfigChangeAlreadyStopped(t *testing.T) {
	err := ensureVMStoppedForConfigChange(context.Background(), nil, 1, freeboxTypes.StoppedStatus, time.Second)
	require.NoError(t, err)

	err = ensureVMStoppedForConfigChange(context.Background(), nil, 1, freeboxTypes.StoppingStatus, time.Second)
	require.NoError(t, err)
}

func TestMinDuration(t *testing.T) {
	assert.Equal(t, 2*time.Second, minDuration(2*time.Second, 5*time.Second))
	assert.Equal(t, 3*time.Second, minDuration(10*time.Second, 3*time.Second))
}
