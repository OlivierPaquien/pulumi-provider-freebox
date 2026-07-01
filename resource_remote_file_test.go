// Integration tests for freebox:downloads:File (RemoteFile).
// Require FREEBOX_APP_ID and FREEBOX_TOKEN.
package main

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/nikolalohinski/free-go/client"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const remoteFileType = "freebox:downloads:File"

// Small file from Terraform provider repo (same as Terraform remote_file test).
const remoteFileTestURL = "https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/refs/heads/main/examples/file-to-download.txt"

func remoteFileURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(remoteFileType), name)
}

func testRemoteFile(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	root := getTestRoot()
	resourceName := "test-" + uuid.New().String()
	destPath := path.Join(root, "VMs", resourceName+".txt")

	t.Run("create and delete", func(t *testing.T) {
		urn := remoteFileURN(resourceName)
		props := property.NewMap(map[string]property.Value{
			"sourceUrl":       property.New(remoteFileTestURL),
			"destinationPath": property.New(destPath),
		})

		createResp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
		require.NoError(t, err)
		require.NotEmpty(t, createResp.ID)
		assert.Equal(t, destPath, createResp.ID)

		// Read back via API
		fileInfo, err := freeboxClient.GetFileInfo(ctx, destPath)
		require.NoError(t, err)
		require.NotNil(t, fileInfo)
		assert.Equal(t, path.Base(destPath), fileInfo.Name)

		// Delete via provider
		err = server.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn, Properties: createResp.Properties})
		require.NoError(t, err)

		// CheckDestroy: file must be gone
		_, err = freeboxClient.GetFileInfo(ctx, destPath)
		assert.ErrorIs(t, err, client.ErrPathNotFound)
	})
}

func getTestRoot() string {
	if r := os.Getenv("FREEBOX_ROOT"); r != "" {
		return r
	}
	return "Freebox"
}
