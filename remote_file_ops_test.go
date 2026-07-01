package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestRemoteFileSourceCount(t *testing.T) {
	assert.Equal(t, 0, remoteFileSourceCount(RemoteFileArgs{}))
	assert.Equal(t, 1, remoteFileSourceCount(RemoteFileArgs{SourceURL: strPtr("http://example.com/file")}))
	assert.Equal(t, 2, remoteFileSourceCount(RemoteFileArgs{
		SourceURL:    strPtr("http://example.com"),
		SourceContent: strPtr("data"),
	}))
}

func TestValidateRemoteFileArgs(t *testing.T) {
	err := validateRemoteFileArgs(RemoteFileArgs{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destinationPath")

	err = validateRemoteFileArgs(RemoteFileArgs{DestinationPath: "/Freebox/file.txt"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of")

	err = validateRemoteFileArgs(RemoteFileArgs{
		DestinationPath: "/Freebox/file.txt",
		SourceURL:       strPtr("http://example.com/file"),
		SourceContent:   strPtr("x"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one of")

	err = validateRemoteFileArgs(RemoteFileArgs{
		DestinationPath: "/Freebox/file.txt",
		SourceURL:       strPtr("http://example.com/file"),
	})
	require.NoError(t, err)
}

func TestSourceChanged(t *testing.T) {
	base := RemoteFileArgs{
		DestinationPath: "/Freebox/a.txt",
		SourceURL:       strPtr("http://example.com/v1"),
	}
	same := RemoteFileArgs{
		DestinationPath: "/Freebox/b.txt",
		SourceURL:       strPtr("http://example.com/v1"),
	}
	changed := RemoteFileArgs{
		DestinationPath: "/Freebox/a.txt",
		SourceURL:       strPtr("http://example.com/v2"),
	}

	assert.False(t, sourceChanged(base, same))
	assert.True(t, sourceChanged(base, changed))
}
