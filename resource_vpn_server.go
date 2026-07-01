package main

import (
	"context"
	"fmt"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

const vpnServerID = "openvpn"

// VpnServer manages the OpenVPN server configuration (singleton per Freebox).
type VpnServer struct{}

type VpnServerArgs struct {
	Enabled    *bool  `pulumi:"enabled,optional"`
	ServerPort *int64 `pulumi:"serverPort,optional"`
	ServerIP   string `pulumi:"serverIp,optional"`
	ServerMask string `pulumi:"serverMask,optional"`
	PushDHCP   *bool  `pulumi:"pushDhcp,optional"`
}

type VpnServerState struct {
	VpnServerArgs
	CA string `pulumi:"ca"`
}

func (VpnServer) Annotate(a infer.Annotator) {
	a.SetToken("vpn", "Server")
}

func vpnServerPayload(args VpnServerArgs, defaults freeboxTypes.OpenVPNServerConfig) freeboxTypes.OpenVPNServerConfig {
	payload := defaults
	if args.Enabled != nil {
		payload.Enabled = *args.Enabled
	}
	if args.ServerPort != nil {
		payload.ServerPort = *args.ServerPort
	}
	if args.ServerIP != "" {
		payload.ServerIP = args.ServerIP
	}
	if args.ServerMask != "" {
		payload.ServerMask = args.ServerMask
	}
	if args.PushDHCP != nil {
		payload.PushDHCP = *args.PushDHCP
	}
	return payload
}

func vpnServerFromConfig(config freeboxTypes.OpenVPNServerConfig) VpnServerState {
	return VpnServerState{
		VpnServerArgs: VpnServerArgs{
			Enabled:    boolPtr(config.Enabled),
			ServerPort: int64Ptr(config.ServerPort),
			ServerIP:   config.ServerIP,
			ServerMask: config.ServerMask,
			PushDHCP:   boolPtr(config.PushDHCP),
		},
		CA: config.CA,
	}
}

func (VpnServer) Create(ctx context.Context, req infer.CreateRequest[VpnServerArgs]) (infer.CreateResponse[VpnServerState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VpnServerState]{}, err
	}

	cfg := providerConfig(ctx)

	current, err := getOpenVPNServerConfigCompat(ctx, cli, cfg)
	if err != nil {
		return infer.CreateResponse[VpnServerState]{}, fmt.Errorf("read OpenVPN server config: %w", err)
	}

	payload := vpnServerPayload(req.Inputs, current)
	if req.DryRun {
		return infer.CreateResponse[VpnServerState]{ID: vpnServerID, Output: vpnServerFromConfig(payload)}, nil
	}

	updated, err := updateOpenVPNServerConfigCompat(ctx, cli, cfg, payload)
	if err != nil {
		return infer.CreateResponse[VpnServerState]{}, fmt.Errorf("configure OpenVPN server: %w", err)
	}

	return infer.CreateResponse[VpnServerState]{
		ID:     vpnServerID,
		Output: vpnServerFromConfig(updated),
	}, nil
}

func (VpnServer) Read(ctx context.Context, req infer.ReadRequest[VpnServerArgs, VpnServerState]) (infer.ReadResponse[VpnServerArgs, VpnServerState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[VpnServerArgs, VpnServerState]{}, err
	}

	cfg := providerConfig(ctx)
	config, err := getOpenVPNServerConfigCompat(ctx, cli, cfg)
	if err != nil {
		return infer.ReadResponse[VpnServerArgs, VpnServerState]{}, fmt.Errorf("read OpenVPN server config: %w", err)
	}

	return infer.ReadResponse[VpnServerArgs, VpnServerState]{State: vpnServerFromConfig(config)}, nil
}

func (VpnServer) Update(ctx context.Context, req infer.UpdateRequest[VpnServerArgs, VpnServerState]) (infer.UpdateResponse[VpnServerState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[VpnServerState]{}, err
	}

	cfg := providerConfig(ctx)

	current, err := getOpenVPNServerConfigCompat(ctx, cli, cfg)
	if err != nil {
		return infer.UpdateResponse[VpnServerState]{}, fmt.Errorf("read OpenVPN server config: %w", err)
	}

	payload := vpnServerPayload(req.Inputs, current)
	if req.DryRun {
		return infer.UpdateResponse[VpnServerState]{Output: vpnServerFromConfig(payload)}, nil
	}

	updated, err := updateOpenVPNServerConfigCompat(ctx, cli, cfg, payload)
	if err != nil {
		return infer.UpdateResponse[VpnServerState]{}, fmt.Errorf("update OpenVPN server: %w", err)
	}

	return infer.UpdateResponse[VpnServerState]{Output: vpnServerFromConfig(updated)}, nil
}

func (VpnServer) Delete(ctx context.Context, req infer.DeleteRequest[VpnServerState]) (infer.DeleteResponse, error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	cfg := providerConfig(ctx)

	current, err := getOpenVPNServerConfigCompat(ctx, cli, cfg)
	if err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("read OpenVPN server config: %w", err)
	}

	payload := vpnServerPayload(req.State.VpnServerArgs, current)
	payload.Enabled = false
	if _, err := updateOpenVPNServerConfigCompat(ctx, cli, cfg, payload); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("disable OpenVPN server: %w", err)
	}
	return infer.DeleteResponse{}, nil
}
