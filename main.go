package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Overridden at release build time via -ldflags "-X main.version=…".
var version = "0.3.11"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "authorize" {
		if err := runAuthorize(version); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	freeboxLog("[freebox] provider freebox starting version %s\n", version)
	p, err := infer.NewProviderBuilder().
		WithDisplayName("Freebox").
		WithDescription("A Pulumi provider for Freebox (port forwarding, VMs, VPN, LAN, DHCP, virtual disks, remote files).").
		WithConfig(infer.Config(Config{})).
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
			"main": "index", "pulumi-freebox": "index",
		}).
		Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p.Run(context.Background(), "freebox", version)
}
