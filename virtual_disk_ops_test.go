package main

import (
	"testing"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVirtualDiskType(t *testing.T) {
	assert.Equal(t, freeboxTypes.QCow2Disk, virtualDiskType(VirtualDiskArgs{}))
	assert.Equal(t, "raw", virtualDiskType(VirtualDiskArgs{Type: "raw"}))
}

func TestVirtualDiskNeedsRecreate(t *testing.T) {
	old := VirtualDiskArgs{Path: "Freebox/disk.qcow2", Type: "qcow2", VirtualSize: 1 << 30}
	newSame := old
	newSame.VirtualSize = 2 << 30

	recreate, err := virtualDiskNeedsRecreate(old, newSame, "")
	require.NoError(t, err)
	assert.False(t, recreate)

	newRaw := old
	newRaw.Type = "raw"
	recreate, err = virtualDiskNeedsRecreate(old, newRaw, "")
	require.NoError(t, err)
	assert.True(t, recreate)

	resizeFrom := "Freebox/source.qcow2"
	oldResize := VirtualDiskArgs{Path: "Freebox/disk.qcow2", VirtualSize: 1 << 30, ResizeFrom: &resizeFrom}
	newResize := oldResize
	recreate, err = virtualDiskNeedsRecreate(oldResize, newResize, `{"sha512":"abc"}`)
	require.NoError(t, err)
	assert.False(t, recreate)

	otherSource := "Freebox/other.qcow2"
	newOther := oldResize
	newOther.ResizeFrom = &otherSource
	recreate, err = virtualDiskNeedsRecreate(oldResize, newOther, `{"sha512":"abc"}`)
	require.NoError(t, err)
	assert.True(t, recreate)

	recreate, err = virtualDiskNeedsRecreate(oldResize, newResize, "")
	require.NoError(t, err)
	assert.False(t, recreate)
}
