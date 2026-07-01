package main

import (
	"context"
	"fmt"

	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

const lanConfigID = "lan_config"

// LanConfig manages the LAN configuration (singleton per Freebox).
type LanConfig struct{}

type LanConfigArgs struct {
	IP          string `pulumi:"ip,optional"`
	Name        string `pulumi:"name,optional"`
	NameDNS     string `pulumi:"nameDns,optional"`
	NameMDNS    string `pulumi:"nameMdns,optional"`
	NameNetBIOS string `pulumi:"nameNetbios,optional"`
	Mode        string `pulumi:"mode,optional"`
}

type LanConfigState struct {
	LanConfigArgs
}

func (LanConfig) Annotate(a infer.Annotator) {
	a.SetToken("lan", "Config")
}

func applyLanConfigArgs(base freeboxTypes.LanConfig, args LanConfigArgs) freeboxTypes.LanConfig {
	if args.IP != "" {
		base.IP = args.IP
	}
	if args.Name != "" {
		base.Name = args.Name
	}
	if args.NameDNS != "" {
		base.NameDNS = args.NameDNS
	}
	if args.NameMDNS != "" {
		base.NameMDNS = args.NameMDNS
	}
	if args.NameNetBIOS != "" {
		base.NameNetBIOS = args.NameNetBIOS
	}
	if args.Mode != "" {
		base.Mode = args.Mode
	}
	return base
}

func lanConfigFromAPI(config freeboxTypes.LanConfig) LanConfigState {
	return LanConfigState{
		LanConfigArgs: LanConfigArgs{
			IP:          config.IP,
			Name:        config.Name,
			NameDNS:     config.NameDNS,
			NameMDNS:    config.NameMDNS,
			NameNetBIOS: config.NameNetBIOS,
			Mode:        config.Mode,
		},
	}
}

func (LanConfig) Create(ctx context.Context, req infer.CreateRequest[LanConfigArgs]) (infer.CreateResponse[LanConfigState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[LanConfigState]{}, err
	}

	current, err := cli.GetLanConfig(ctx)
	if err != nil {
		return infer.CreateResponse[LanConfigState]{}, fmt.Errorf("read LAN config: %w", err)
	}

	payload := applyLanConfigArgs(current, req.Inputs)
	if req.DryRun {
		return infer.CreateResponse[LanConfigState]{ID: lanConfigID, Output: lanConfigFromAPI(payload)}, nil
	}

	updated, err := cli.UpdateLanConfig(ctx, payload)
	if err != nil {
		return infer.CreateResponse[LanConfigState]{}, fmt.Errorf("update LAN config: %w", err)
	}

	return infer.CreateResponse[LanConfigState]{ID: lanConfigID, Output: lanConfigFromAPI(updated)}, nil
}

func (LanConfig) Read(ctx context.Context, req infer.ReadRequest[LanConfigArgs, LanConfigState]) (infer.ReadResponse[LanConfigArgs, LanConfigState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[LanConfigArgs, LanConfigState]{}, err
	}

	config, err := cli.GetLanConfig(ctx)
	if err != nil {
		return infer.ReadResponse[LanConfigArgs, LanConfigState]{}, fmt.Errorf("read LAN config: %w", err)
	}

	return infer.ReadResponse[LanConfigArgs, LanConfigState]{State: lanConfigFromAPI(config)}, nil
}

func (LanConfig) Update(ctx context.Context, req infer.UpdateRequest[LanConfigArgs, LanConfigState]) (infer.UpdateResponse[LanConfigState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[LanConfigState]{}, err
	}

	current, err := cli.GetLanConfig(ctx)
	if err != nil {
		return infer.UpdateResponse[LanConfigState]{}, fmt.Errorf("read LAN config: %w", err)
	}

	payload := applyLanConfigArgs(current, req.Inputs)
	if req.DryRun {
		return infer.UpdateResponse[LanConfigState]{Output: lanConfigFromAPI(payload)}, nil
	}

	updated, err := cli.UpdateLanConfig(ctx, payload)
	if err != nil {
		return infer.UpdateResponse[LanConfigState]{}, fmt.Errorf("update LAN config: %w", err)
	}

	return infer.UpdateResponse[LanConfigState]{Output: lanConfigFromAPI(updated)}, nil
}

func (LanConfig) Delete(ctx context.Context, req infer.DeleteRequest[LanConfigState]) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, nil
}
