package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

const version = "0.1.1"

func main() {
	freeboxLog("[freebox] provider freebox starting version %s\n", version)
	p, err := infer.NewProviderBuilder().
		WithDisplayName("Freebox").
		WithDescription("A Pulumi provider for Freebox (port forwarding, VMs, virtual disks, remote files).").
		WithConfig(infer.Config(Config{})).
		WithResources(
			infer.Resource(PortForwarding{}),
			infer.Resource(VirtualDisk{}),
			infer.Resource(VirtualMachine{}),
			infer.Resource(RemoteFile{}),
		).
		WithFunctions(
			infer.Function(GetApiVersion{}),
			infer.Function(GetVirtualDisk{}),
		).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"main": "index", "pulumi-provider-freebox": "index",
		}).
		Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p.Run(context.Background(), "freebox", version)
}
