package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-go-provider/infer"
)

const version = "0.1.0"

func main() {
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
		Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p.Run(context.Background(), "freebox", version)
}
