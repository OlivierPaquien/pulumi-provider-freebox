// Package main: test suite for the Freebox provider.
// Integration tests require FREEBOX_APP_ID and FREEBOX_TOKEN (and optionally FREEBOX_ENDPOINT, FREEBOX_VERSION).
package main

import (
	"context"
	"os"
	"testing"

	"github.com/blang/semver"
	"github.com/nikolalohinski/free-go/client"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	if os.Getenv("FREEBOX_TOKEN") == "" {
		t.Skip("FREEBOX_TOKEN not set, skipping integration tests")
	}
	// Run resource tests
	t.Run("PortForwarding", testPortForwarding)
	t.Run("RemoteFile", testRemoteFile)
	t.Run("VpnServer", testVpnServer)
	t.Run("VpnUser", testVpnUser)
	if os.Getenv("FREEBOX_TEST_DISK_PATH") != "" || os.Getenv("FREEBOX_TEST_VM_ID") != "" {
		t.Run("VirtualMachinePower", testVirtualMachinePower)
	}
	if os.Getenv("FREEBOX_TEST_DISK_PATH") != "" {
		t.Run("VirtualMachine", testVirtualMachine)
	}
}

// freeboxTestConfig returns provider config from environment (same as Terraform suite).
func freeboxTestConfig() Config {
	endpoint := os.Getenv("FREEBOX_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://mafreebox.freebox.fr"
	}
	version := os.Getenv("FREEBOX_VERSION")
	if version == "" {
		version = "latest"
	}
	appID := os.Getenv("FREEBOX_APP_ID")
	if appID == "" {
		appID = "terraform-provider-freebox"
	}
	return Config{
		Endpoint:   endpoint,
		APIVersion: version,
		AppID:      appID,
		Token:      os.Getenv("FREEBOX_TOKEN"),
	}
}

// newTestServer builds the provider with the given config and returns an integration Server.
func newTestServer(t *testing.T, cfg Config) integration.Server {
	p, err := infer.NewProviderBuilder().
		WithDisplayName("Freebox").
		WithDescription("A Pulumi provider for Freebox.").
		WithConfig(infer.Config(cfg)).
		WithResources(
			infer.Resource(PortForwarding{}),
			infer.Resource(VirtualDisk{}),
			infer.Resource(VirtualMachine{}),
			infer.Resource(VirtualMachinePower{}),
			infer.Resource(DHCPStaticLease{}),
			infer.Resource(RemoteFile{}),
			infer.Resource(VpnServer{}),
			infer.Resource(VpnUser{}),
			infer.Resource(LanConfig{}),
		).
		WithFunctions(
			infer.Function(GetApiVersion{}),
			infer.Function(GetVirtualDisk{}),
			infer.Function(GetDhcpLease{}),
			infer.Function(GetDhcpLeases{}),
			infer.Function(GetLanConfig{}),
			infer.Function(GetLanInterfaces{}),
			infer.Function(GetLanInterfaceHosts{}),
			infer.Function(GetLanInterfaceHost{}),
			infer.Function(GetVmDistributions{}),
			infer.Function(GetSystemInfo{}),
		).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"main": "index", "pulumi-provider-freebox": "index",
		}).
		Build()
	require.NoError(t, err)

	s, err := integration.NewServer(
		context.Background(),
		"freebox",
		semver.MustParse("0.2.0"),
		integration.WithProvider(p),
	)
	require.NoError(t, err)
	return s
}

// newFreeboxClient builds a free-go client from env (for CheckDestroy-style checks).
func newFreeboxClient(t *testing.T, ctx context.Context) client.Client {
	cfg := freeboxTestConfig()
	require.NotEmpty(t, cfg.AppID, "FREEBOX_APP_ID required")
	require.NotEmpty(t, cfg.Token, "FREEBOX_TOKEN required")

	c, err := client.New(cfg.Endpoint, cfg.APIVersion)
	require.NoError(t, err)
	c = c.WithAppID(cfg.AppID).WithPrivateToken(cfg.Token)
	_, err = c.Login(ctx)
	require.NoError(t, err)
	return c
}
