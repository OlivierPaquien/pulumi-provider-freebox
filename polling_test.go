package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPollingSpecWithDefaults(t *testing.T) {
	var nilSpec *PollingSpec
	out := nilSpec.withDefaults(time.Second, time.Minute)
	assert.Equal(t, time.Second, out.Interval)
	assert.Equal(t, time.Minute, out.Timeout)

	custom := &PollingSpec{Interval: 5 * time.Second}
	out = custom.withDefaults(time.Second, time.Minute)
	assert.Equal(t, 5*time.Second, out.Interval)
	assert.Equal(t, time.Minute, out.Timeout)
}

func TestRemoteFilePollingResolved(t *testing.T) {
	def := defaultRemoteFilePolling()
	var nilPolling *RemoteFilePolling
	resolved := nilPolling.resolved()
	assert.Equal(t, def.Download.Interval, resolved.Download.Interval)
	assert.Equal(t, def.Download.Timeout, resolved.Download.Timeout)

	empty := &RemoteFilePolling{}
	resolved = empty.resolved()
	assert.Equal(t, def.Upload.Timeout, resolved.Upload.Timeout)

	partial := &RemoteFilePolling{
		Download: &PollingSpec{Timeout: 10 * time.Minute},
	}
	resolved = partial.resolved()
	assert.Equal(t, 10*time.Minute, resolved.Download.Timeout)
	assert.Equal(t, time.Duration(0), resolved.Download.Interval)
	assert.Equal(t, def.Upload.Timeout, resolved.Upload.Timeout)
}

func TestVirtualDiskPollingResolved(t *testing.T) {
	def := defaultVirtualDiskPolling()
	var nilPolling *VirtualDiskPolling
	resolved := nilPolling.resolved()
	assert.Equal(t, def.Create.Interval, resolved.Create.Interval)

	partial := &VirtualDiskPolling{
		Resize: &PollingSpec{Interval: 3 * time.Second},
	}
	resolved = partial.resolved()
	assert.Equal(t, 3*time.Second, resolved.Resize.Interval)
	assert.Equal(t, time.Duration(0), resolved.Resize.Timeout)
	assert.Equal(t, def.Copy.Interval, resolved.Copy.Interval)
}
