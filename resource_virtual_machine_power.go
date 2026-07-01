package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VirtualMachinePower manages VM power state independently from VM configuration.
type VirtualMachinePower struct{}

type VirtualMachinePowerArgs struct {
	VmId        int64                        `pulumi:"vmId"`
	PowerState  string                       `pulumi:"powerState"`
	KillTimeout int64                        `pulumi:"killTimeout,optional"` // seconds; default 30
	Timeouts    *VirtualMachinePowerTimeouts `pulumi:"timeouts,optional"`
}

type VirtualMachinePowerState struct {
	VirtualMachinePowerArgs
}

func (args *VirtualMachinePowerArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.VmId, "Identifier of the virtual machine.")
	a.Describe(&args.PowerState, "Desired power state: running or stopped.")
	a.Describe(&args.KillTimeout, "Seconds to wait for graceful shutdown before force kill (default 30). Deprecated: use timeouts.kill.")
	a.Describe(&args.Timeouts, "Timeouts for power operations.")
}

func (st *VirtualMachinePowerState) Annotate(a infer.Annotator) {
	a.Describe(&st.VmId, "Identifier of the virtual machine.")
}

func (VirtualMachinePower) Annotate(a infer.Annotator) {
	a.SetToken("virtual", "MachinePower")
}

func powerKillTimeout(args VirtualMachinePowerArgs) time.Duration {
	return args.Timeouts.killTimeout(args.KillTimeout)
}

func applyPowerState(ctx context.Context, cli client.Client, vmID int64, desired string, killTimeout time.Duration) error {
	vm, err := cli.GetVirtualMachine(ctx, vmID)
	if err != nil {
		return fmt.Errorf("get virtual machine: %w", err)
	}

	switch desired {
	case freeboxTypes.RunningStatus:
		if vm.Status == freeboxTypes.RunningStatus || vm.Status == freeboxTypes.StartingStatus {
			return nil
		}
		if _, err := vmStart(ctx, cli, vmID); err != nil {
			return fmt.Errorf("start virtual machine: %w", err)
		}
	case freeboxTypes.StoppedStatus:
		if vm.Status == freeboxTypes.StoppedStatus || vm.Status == freeboxTypes.StoppingStatus {
			return nil
		}
		if _, err := vmStop(ctx, cli, vmID, killTimeout); err != nil {
			return fmt.Errorf("stop virtual machine: %w", err)
		}
	default:
		return fmt.Errorf("unsupported power state %q", desired)
	}
	return nil
}

func normalizePowerState(status string) string {
	switch status {
	case freeboxTypes.RunningStatus, freeboxTypes.StartingStatus:
		return freeboxTypes.RunningStatus
	case freeboxTypes.StoppedStatus, freeboxTypes.StoppingStatus:
		return freeboxTypes.StoppedStatus
	default:
		return status
	}
}

func (VirtualMachinePower) Create(ctx context.Context, req infer.CreateRequest[VirtualMachinePowerArgs]) (infer.CreateResponse[VirtualMachinePowerState], error) {
	if req.Inputs.PowerState != freeboxTypes.RunningStatus && req.Inputs.PowerState != freeboxTypes.StoppedStatus {
		return infer.CreateResponse[VirtualMachinePowerState]{}, fmt.Errorf("powerState must be %q or %q", freeboxTypes.RunningStatus, freeboxTypes.StoppedStatus)
	}

	id := strconv.FormatInt(req.Inputs.VmId, 10)
	if req.DryRun {
		return infer.CreateResponse[VirtualMachinePowerState]{
			ID: id,
			Output: VirtualMachinePowerState{
				VirtualMachinePowerArgs: req.Inputs,
			},
		}, nil
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VirtualMachinePowerState]{}, err
	}

	if err := applyPowerState(ctx, cli, req.Inputs.VmId, req.Inputs.PowerState, powerKillTimeout(req.Inputs)); err != nil {
		return infer.CreateResponse[VirtualMachinePowerState]{}, err
	}

	return infer.CreateResponse[VirtualMachinePowerState]{
		ID: id,
		Output: VirtualMachinePowerState{
			VirtualMachinePowerArgs: req.Inputs,
		},
	}, nil
}

func (VirtualMachinePower) Read(ctx context.Context, req infer.ReadRequest[VirtualMachinePowerArgs, VirtualMachinePowerState]) (infer.ReadResponse[VirtualMachinePowerArgs, VirtualMachinePowerState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[VirtualMachinePowerArgs, VirtualMachinePowerState]{}, err
	}

	vm, err := cli.GetVirtualMachine(ctx, req.State.VmId)
	if err != nil {
		return infer.ReadResponse[VirtualMachinePowerArgs, VirtualMachinePowerState]{}, nil
	}

	state := VirtualMachinePowerState{
		VirtualMachinePowerArgs: VirtualMachinePowerArgs{
			VmId:        req.State.VmId,
			PowerState:  normalizePowerState(vm.Status),
			KillTimeout: req.State.KillTimeout,
		},
	}
	return infer.ReadResponse[VirtualMachinePowerArgs, VirtualMachinePowerState]{State: state}, nil
}

func (VirtualMachinePower) Update(ctx context.Context, req infer.UpdateRequest[VirtualMachinePowerArgs, VirtualMachinePowerState]) (infer.UpdateResponse[VirtualMachinePowerState], error) {
	if req.Inputs.PowerState != freeboxTypes.RunningStatus && req.Inputs.PowerState != freeboxTypes.StoppedStatus {
		return infer.UpdateResponse[VirtualMachinePowerState]{}, fmt.Errorf("powerState must be %q or %q", freeboxTypes.RunningStatus, freeboxTypes.StoppedStatus)
	}

	if req.DryRun {
		return infer.UpdateResponse[VirtualMachinePowerState]{
			Output: VirtualMachinePowerState{VirtualMachinePowerArgs: req.Inputs},
		}, nil
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[VirtualMachinePowerState]{}, err
	}

	if err := applyPowerState(ctx, cli, req.Inputs.VmId, req.Inputs.PowerState, powerKillTimeout(req.Inputs)); err != nil {
		return infer.UpdateResponse[VirtualMachinePowerState]{}, err
	}

	return infer.UpdateResponse[VirtualMachinePowerState]{
		Output: VirtualMachinePowerState{VirtualMachinePowerArgs: req.Inputs},
	}, nil
}

func (VirtualMachinePower) Delete(ctx context.Context, req infer.DeleteRequest[VirtualMachinePowerState]) (infer.DeleteResponse, error) {
	if req.State.PowerState == freeboxTypes.StoppedStatus {
		return infer.DeleteResponse{}, nil
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	if err := applyPowerState(ctx, cli, req.State.VmId, freeboxTypes.StoppedStatus, powerKillTimeout(req.State.VirtualMachinePowerArgs)); err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, nil
}
