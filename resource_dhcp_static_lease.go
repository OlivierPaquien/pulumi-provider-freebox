package main

import (
	"context"
	"fmt"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// DHCPStaticLease manages a DHCP static lease on the Freebox.
type DHCPStaticLease struct{}

type DHCPStaticLeaseArgs struct {
	Mac      string `pulumi:"mac"`
	IP       string `pulumi:"ip"`
	Comment  string `pulumi:"comment,optional"`
	Hostname string `pulumi:"hostname,optional"`
}

type DHCPStaticLeaseState struct {
	DHCPStaticLeaseArgs
}

func (args *DHCPStaticLeaseArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Mac, "MAC address of the host.")
	a.Describe(&args.IP, "IPv4 address to assign to the host.")
	a.Describe(&args.Comment, "Optional comment for the lease.")
	a.Describe(&args.Hostname, "Optional hostname for the lease.")
}

func (st *DHCPStaticLeaseState) Annotate(a infer.Annotator) {
	a.Describe(&st.Mac, "MAC address of the host (resource identifier).")
}

func (DHCPStaticLease) Annotate(a infer.Annotator) {
	a.SetToken("dhcp", "StaticLease")
}

func (DHCPStaticLease) Create(ctx context.Context, req infer.CreateRequest[DHCPStaticLeaseArgs]) (infer.CreateResponse[DHCPStaticLeaseState], error) {
	if err := validateDHCPStaticLeaseArgs(req.Inputs.Mac, req.Inputs.IP); err != nil {
		return infer.CreateResponse[DHCPStaticLeaseState]{}, err
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[DHCPStaticLeaseState]{}, err
	}

	mac := normalizeMAC(req.Inputs.Mac)
	payload := freeboxTypes.DHCPStaticLeasePayload{
		Mac: mac,
		IP:  req.Inputs.IP,
	}
	if req.Inputs.Comment != "" {
		payload.Comment = req.Inputs.Comment
	}
	if req.Inputs.Hostname != "" {
		payload.Hostname = req.Inputs.Hostname
	}

	if req.DryRun {
		return infer.CreateResponse[DHCPStaticLeaseState]{
			ID: mac,
			Output: DHCPStaticLeaseState{
				DHCPStaticLeaseArgs: req.Inputs,
			},
		}, nil
	}

	if err := ensureDHCPStaticLease(ctx, cli, payload); err != nil {
		return infer.CreateResponse[DHCPStaticLeaseState]{}, err
	}

	lease, err := cli.GetDHCPStaticLease(ctx, mac)
	if err != nil {
		return infer.CreateResponse[DHCPStaticLeaseState]{}, fmt.Errorf("read DHCP static lease: %w", err)
	}

	state := DHCPStaticLeaseState{
		DHCPStaticLeaseArgs: dhcpLeaseStateFromAPI(lease, req.Inputs.Hostname, req.Inputs.Mac),
	}

	return infer.CreateResponse[DHCPStaticLeaseState]{
		ID:     normalizeMAC(mac),
		Output: state,
	}, nil
}

func (DHCPStaticLease) Read(ctx context.Context, req infer.ReadRequest[DHCPStaticLeaseArgs, DHCPStaticLeaseState]) (infer.ReadResponse[DHCPStaticLeaseArgs, DHCPStaticLeaseState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[DHCPStaticLeaseArgs, DHCPStaticLeaseState]{}, err
	}

	mac := normalizeMAC(req.State.Mac)
	lease, err := cli.GetDHCPStaticLease(ctx, mac)
	if err != nil {
		return infer.ReadResponse[DHCPStaticLeaseArgs, DHCPStaticLeaseState]{}, nil
	}

	state := DHCPStaticLeaseState{
		DHCPStaticLeaseArgs: dhcpLeaseStateFromAPI(lease, req.State.Hostname, req.State.Mac),
	}
	return infer.ReadResponse[DHCPStaticLeaseArgs, DHCPStaticLeaseState]{State: state}, nil
}

func (DHCPStaticLease) Update(ctx context.Context, req infer.UpdateRequest[DHCPStaticLeaseArgs, DHCPStaticLeaseState]) (infer.UpdateResponse[DHCPStaticLeaseState], error) {
	if err := validateDHCPStaticLeaseArgs(req.Inputs.Mac, req.Inputs.IP); err != nil {
		return infer.UpdateResponse[DHCPStaticLeaseState]{}, err
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[DHCPStaticLeaseState]{}, err
	}

	mac := normalizeMAC(req.Inputs.Mac)
	payload := freeboxTypes.DHCPStaticLeasePayload{
		Mac: mac,
		IP:  req.Inputs.IP,
	}
	if req.Inputs.Comment != "" {
		payload.Comment = req.Inputs.Comment
	}
	if req.Inputs.Hostname != "" {
		payload.Hostname = req.Inputs.Hostname
	}

	if req.DryRun {
		return infer.UpdateResponse[DHCPStaticLeaseState]{
			Output: DHCPStaticLeaseState{DHCPStaticLeaseArgs: req.Inputs},
		}, nil
	}

	if err := ensureDHCPStaticLease(ctx, cli, payload); err != nil {
		return infer.UpdateResponse[DHCPStaticLeaseState]{}, err
	}

	lease, err := cli.GetDHCPStaticLease(ctx, mac)
	if err != nil {
		return infer.UpdateResponse[DHCPStaticLeaseState]{}, fmt.Errorf("read DHCP static lease: %w", err)
	}

	state := DHCPStaticLeaseState{
		DHCPStaticLeaseArgs: dhcpLeaseStateFromAPI(lease, req.Inputs.Hostname, req.Inputs.Mac),
	}
	return infer.UpdateResponse[DHCPStaticLeaseState]{Output: state}, nil
}

func (DHCPStaticLease) Delete(ctx context.Context, req infer.DeleteRequest[DHCPStaticLeaseState]) (infer.DeleteResponse, error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if err := deleteDHCPStaticLeaseIfExists(ctx, cli, req.State.Mac); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}
