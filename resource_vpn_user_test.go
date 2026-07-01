// Integration tests for freebox:vpn:User.
// Require FREEBOX_APP_ID and FREEBOX_TOKEN.
package main

import (
	"context"
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

func vpnUserURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), name)
}

func testVpnUser(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	beforeVPN, err := getOpenVPNServerConfigCompat(ctx, freeboxClient, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = updateOpenVPNServerConfigCompat(ctx, freeboxClient, cfg, beforeVPN)
	})

	login := "pvpn-" + uuid.New().String()[:8]
	password := "Pw1-" + uuid.New().String()[:8]
	description := "pulumi-provider-freebox integration test"

	t.Run("create, update and delete", func(t *testing.T) {
		urn := vpnUserURN(login)
		props := property.NewMap(map[string]property.Value{
			"login":       property.New(login),
			"password":    property.New(password),
			"description": property.New(description),
		})

		createResp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
		if err != nil {
			_ = freeboxClient.DeleteVPNUser(ctx, login)
		}
		require.NoError(t, err)
		require.Equal(t, login, createResp.ID)
		assert.NotEmpty(t, createResp.Properties.Get("ovpnConfig").AsString())
		assert.Equal(t, description, createResp.Properties.Get("description").AsString())

		user, err := freeboxClient.GetVPNUser(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, login, user.Login)

		newPassword := "Nw2-" + uuid.New().String()[:8]
		updateResp, err := server.Update(p.UpdateRequest{
			ID:    login,
			Urn:   urn,
			State: createResp.Properties,
			Inputs: property.NewMap(map[string]property.Value{
				"login":       property.New(login),
				"password":    property.New(newPassword),
				"description": property.New(description + " updated"),
			}),
			OldInputs: createResp.Properties,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, updateResp.Properties.Get("ovpnConfig").AsString())
		assert.Equal(t, description+" updated", updateResp.Properties.Get("description").AsString())
		assert.NotEqual(t,
			createResp.Properties.Get("ovpnConfig").AsString(),
			updateResp.Properties.Get("ovpnConfig").AsString(),
		)

		err = server.Delete(p.DeleteRequest{ID: login, Urn: urn, Properties: updateResp.Properties})
		require.NoError(t, err)

		_, err = freeboxClient.GetVPNUser(ctx, login)
		assert.ErrorIs(t, err, client.ErrVPNUserNotFound)
	})
}
