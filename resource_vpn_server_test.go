// Integration tests for freebox:vpn:Server.
// Require FREEBOX_APP_ID and FREEBOX_TOKEN.
// Write tests (create/delete) require FREEBOX_TEST_VPN_SERVER=1 and restore prior config on cleanup.
package main

import (
	"context"
	"os"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func vpnServerURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(vpnServerType), name)
}

func testVpnServer(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	t.Run("read", func(t *testing.T) {
		urn := vpnServerURN("read")
		resp, err := server.Read(p.ReadRequest{
			ID:         vpnServerID,
			Urn:        urn,
			Properties: property.NewMap(map[string]property.Value{}),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Properties)
		assert.Contains(t, []bool{true, false}, resp.Properties.Get("enabled").AsBool())
		if ca := resp.Properties.Get("ca").AsString(); ca != "" {
			assert.Contains(t, ca, "BEGIN CERTIFICATE")
		}
	})

	if os.Getenv("FREEBOX_TEST_VPN_SERVER") != "1" {
		t.Log("skip VpnServer write tests: set FREEBOX_TEST_VPN_SERVER=1 to enable create/delete")
		return
	}

	t.Run("create and delete", func(t *testing.T) {
		before, err := getOpenVPNServerConfigCompat(ctx, freeboxClient, cfg)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = updateOpenVPNServerConfigCompat(ctx, freeboxClient, cfg, before)
		})

		urn := vpnServerURN("write")
		enabled := before.Enabled
		createResp, err := server.Create(p.CreateRequest{
			Urn: urn,
			Properties: property.NewMap(map[string]property.Value{
				"enabled": property.New(enabled),
			}),
		})
		require.NoError(t, err)
		require.Equal(t, vpnServerID, createResp.ID)
		assert.Equal(t, property.New(enabled), createResp.Properties.Get("enabled"))

		err = server.Delete(p.DeleteRequest{
			ID:         vpnServerID,
			Urn:        urn,
			Properties: createResp.Properties,
		})
		require.NoError(t, err)

		afterDelete, err := getOpenVPNServerConfigCompat(ctx, freeboxClient, cfg)
		require.NoError(t, err)
		assert.False(t, afterDelete.Enabled)
	})
}
