package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VpnUser manages an OpenVPN user account on the Freebox.
type VpnUser struct{}

type VpnUserArgs struct {
	Login       string `pulumi:"login"`
	Password    string `pulumi:"password"`
	Description string `pulumi:"description,optional"`
}

type VpnUserState struct {
	VpnUserArgs
	OvpnConfig string `pulumi:"ovpnConfig"`
}

func (VpnUser) Annotate(a infer.Annotator) {
	a.SetToken("vpn", "User")
}

func validateVPNUserPassword(password string) error {
	n := len(password)
	if n < 8 || n > 32 {
		return fmt.Errorf("VPN password must be between 8 and 32 characters (got %d)", n)
	}
	return nil
}

func createVPNUserCompat(ctx context.Context, cli client.Client, args VpnUserArgs) (freeboxTypes.VPNUser, error) {
	payload := freeboxTypes.VPNUserPayload{
		Login:    args.Login,
		Password: args.Password,
	}
	if !usesV4VPNAPI(ctx, cli) {
		payload.Description = args.Description
	}
	return cli.CreateVPNUser(ctx, payload)
}

func vpnUserDescriptionFromState(state VpnUserState, inputs VpnUserArgs, fromAPI string) string {
	if inputs.Description != "" {
		return inputs.Description
	}
	if state.Description != "" {
		return state.Description
	}
	return fromAPI
}

func (VpnUser) Create(ctx context.Context, req infer.CreateRequest[VpnUserArgs]) (infer.CreateResponse[VpnUserState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VpnUserState]{}, err
	}

	if req.DryRun {
		return infer.CreateResponse[VpnUserState]{
			ID:     req.Inputs.Login,
			Output: VpnUserState{VpnUserArgs: req.Inputs},
		}, nil
	}

	if err := validateVPNUserPassword(req.Inputs.Password); err != nil {
		return infer.CreateResponse[VpnUserState]{}, err
	}

	user, err := createVPNUserCompat(ctx, cli, req.Inputs)
	if err != nil {
		return infer.CreateResponse[VpnUserState]{}, fmt.Errorf("create VPN user: %w", err)
	}

	cfg := providerConfig(ctx)
	ovpn, err := getVPNUserClientConfigCompat(ctx, cli, cfg, user.Login)
	if err != nil {
		return infer.CreateResponse[VpnUserState]{}, fmt.Errorf("get VPN client config: %w", err)
	}

	return infer.CreateResponse[VpnUserState]{
		ID:     user.Login,
		Output: VpnUserState{VpnUserArgs: req.Inputs, OvpnConfig: ovpn},
	}, nil
}

func (VpnUser) Read(ctx context.Context, req infer.ReadRequest[VpnUserArgs, VpnUserState]) (infer.ReadResponse[VpnUserArgs, VpnUserState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[VpnUserArgs, VpnUserState]{}, err
	}

	user, err := cli.GetVPNUser(ctx, req.ID)
	if err != nil {
		if errors.Is(err, client.ErrVPNUserNotFound) {
			return infer.ReadResponse[VpnUserArgs, VpnUserState]{}, nil
		}
		return infer.ReadResponse[VpnUserArgs, VpnUserState]{}, fmt.Errorf("read VPN user: %w", err)
	}

	ovpn, err := getVPNUserClientConfigCompat(ctx, cli, providerConfig(ctx), user.Login)
	if err != nil {
		return infer.ReadResponse[VpnUserArgs, VpnUserState]{}, fmt.Errorf("get VPN client config: %w", err)
	}

	state := VpnUserState{
		VpnUserArgs: VpnUserArgs{
			Login:       user.Login,
			Password:    req.State.Password,
			Description: vpnUserDescriptionFromState(req.State, req.Inputs, user.Description),
		},
		OvpnConfig: ovpn,
	}
	return infer.ReadResponse[VpnUserArgs, VpnUserState]{State: state}, nil
}

func (VpnUser) Update(ctx context.Context, req infer.UpdateRequest[VpnUserArgs, VpnUserState]) (infer.UpdateResponse[VpnUserState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[VpnUserState]{}, err
	}

	if req.DryRun {
		return infer.UpdateResponse[VpnUserState]{
			Output: VpnUserState{VpnUserArgs: req.Inputs, OvpnConfig: req.State.OvpnConfig},
		}, nil
	}

	if err := validateVPNUserPassword(req.Inputs.Password); err != nil {
		return infer.UpdateResponse[VpnUserState]{}, err
	}

	cfg := providerConfig(ctx)
	user, err := updateVPNUserCompat(ctx, cli, cfg, req.Inputs.Login, req.Inputs.Password)
	if err != nil {
		if errors.Is(err, client.ErrVPNUserNotFound) {
			return infer.UpdateResponse[VpnUserState]{}, fmt.Errorf("VPN user %q not found", req.Inputs.Login)
		}
		return infer.UpdateResponse[VpnUserState]{}, fmt.Errorf("update VPN user: %w", err)
	}

	ovpn, err := getVPNUserClientConfigCompat(ctx, cli, cfg, user.Login)
	if err != nil {
		return infer.UpdateResponse[VpnUserState]{}, fmt.Errorf("get VPN client config: %w", err)
	}

	state := VpnUserState{
		VpnUserArgs: VpnUserArgs{
			Login:       user.Login,
			Password:    req.Inputs.Password,
			Description: req.Inputs.Description,
		},
		OvpnConfig: ovpn,
	}
	return infer.UpdateResponse[VpnUserState]{Output: state}, nil
}

func (VpnUser) Delete(ctx context.Context, req infer.DeleteRequest[VpnUserState]) (infer.DeleteResponse, error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if err := cli.DeleteVPNUser(ctx, req.State.Login); err != nil && !errors.Is(err, client.ErrVPNUserNotFound) {
		return infer.DeleteResponse{}, fmt.Errorf("delete VPN user: %w", err)
	}
	return infer.DeleteResponse{}, nil
}
