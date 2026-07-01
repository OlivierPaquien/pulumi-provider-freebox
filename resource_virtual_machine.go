package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type vmStateEvent struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// VirtualMachine resource: manages a VM on the Freebox.
type VirtualMachine struct{}

type VirtualMachineArgs struct {
	Name              string                  `pulumi:"name"`
	DiskPath          string                  `pulumi:"diskPath"`
	DiskType          string                  `pulumi:"diskType"`
	Memory            int64                   `pulumi:"memory"`
	VCPUs             int64                   `pulumi:"vcpus"`
	Status            string                  `pulumi:"status,optional"`
	CDPath            string                  `pulumi:"cdPath,optional"`
	OS                string                  `pulumi:"os,optional"`
	EnableScreen      bool                    `pulumi:"enableScreen,optional"`
	EnableCloudInit   bool                    `pulumi:"enableCloudinit,optional"`
	CloudInitUserData string                  `pulumi:"cloudinitUserdata,optional"`
	CloudInitHostname string                  `pulumi:"cloudinitHostname,optional"`
	BindUSBPorts      []string                `pulumi:"bindUsbPorts,optional"`
	Timeouts          *VirtualMachineTimeouts `pulumi:"timeouts,optional"`
}

// NetworkingBind holds one network bind (interface + IPv4/IPv6) for the VM.
type NetworkingBind struct {
	Interface string   `pulumi:"interface"`
	Ipv4      string   `pulumi:"ipv4"`
	Ipv6      []string `pulumi:"ipv6,optional"`
}

type VirtualMachineState struct {
	VirtualMachineArgs
	ID         int64                   `pulumi:"vmId"` // id réservé par Pulumi
	Mac        string                  `pulumi:"mac"`
	Status     string                  `pulumi:"status"`
	Networking []NetworkingBind        `pulumi:"networking,optional"`
	Ipv4       string                  `pulumi:"ipv4,optional"` // First IPv4 (convenience, same as networking[0].ipv4)
	Timeouts   *VirtualMachineTimeouts `pulumi:"timeouts,optional"`
}

func (args *VirtualMachineArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Name, "Name of the VM.")
	a.Describe(&args.DiskPath, "Path to the disk image.")
	a.Describe(&args.DiskType, "Disk type: qcow2 or raw.")
	a.Describe(&args.Memory, "Memory in MB.")
	a.Describe(&args.VCPUs, "Number of vCPUs.")
	a.Describe(&args.Status, "Desired state: running or stopped. Default running.")
	a.Describe(&args.CDPath, "Path to CD/ISO image.")
	a.Describe(&args.OS, "OS type (for icon).")
	a.Describe(&args.EnableScreen, "Enable VNC screen.")
	a.Describe(&args.EnableCloudInit, "Enable cloud-init.")
	a.Describe(&args.CloudInitUserData, "Cloud-init user-data.")
	a.Describe(&args.CloudInitHostname, "Cloud-init hostname.")
	a.Describe(&args.BindUSBPorts, "USB ports to bind.")
	a.Describe(&args.Timeouts, "Operation timeouts (create, update, read, delete, kill, networking).")
}

func (st *VirtualMachineState) Annotate(a infer.Annotator) {
	a.Describe(&st.ID, "Freebox API VM ID.")
	a.Describe(&st.Mac, "MAC address of the VM.")
	a.Describe(&st.Status, "Current VM status (running/stopped).")
	a.Describe(&st.Networking, "Network binds (interface, IPv4, IPv6) for the VM.")
	a.Describe(&st.Ipv4, "First IPv4 address of the VM on the LAN (when running).")
}

func (VirtualMachine) Annotate(a infer.Annotator) {
	// VirtualMachine est une struct vide ; l'Annotator reçoit ce type uniquement. Ne pas appeler Describe.
	a.SetToken("virtual", "Machine")
}

func toPayload(in VirtualMachineArgs) freeboxTypes.VirtualMachinePayload {
	p := freeboxTypes.VirtualMachinePayload{
		Name:              in.Name,
		DiskPath:          freeboxTypes.Base64Path(in.DiskPath),
		DiskType:          in.DiskType,
		Memory:            in.Memory,
		VCPUs:             in.VCPUs,
		CDPath:            freeboxTypes.Base64Path(in.CDPath),
		EnableScreen:      in.EnableScreen,
		EnableCloudInit:   in.EnableCloudInit,
		CloudInitUserData: in.CloudInitUserData,
		CloudHostName:     in.CloudInitHostname,
		BindUSBPorts:      in.BindUSBPorts,
	}
	if in.OS != "" {
		p.OS = in.OS
	}
	return p
}

func desiredVMStatus(in VirtualMachineArgs) string {
	status := in.Status
	if status == "" {
		status = freeboxTypes.RunningStatus
	}
	return status
}

func fillVMNetworking(ctx context.Context, cli client.Client, vm freeboxTypes.VirtualMachine, state *VirtualMachineState, networkingTimeout time.Duration) {
	if state.Status != freeboxTypes.RunningStatus {
		return
	}
	binds, err := getNetworkBinds(ctx, cli, vm, networkingTimeout)
	if err != nil {
		freeboxLog("[freebox] VM vmId=%d: get network binds: %v\n", vm.ID, err)
		return
	}
	if len(binds) > 0 {
		state.Networking = binds
		state.Ipv4 = binds[0].Ipv4
	}
}

func applyVMDesiredStatus(ctx context.Context, cli client.Client, vmID int64, desiredStatus string, killTimeout time.Duration) (string, error) {
	switch desiredStatus {
	case freeboxTypes.RunningStatus:
		return vmStart(ctx, cli, vmID)
	case freeboxTypes.StoppedStatus:
		vm, err := cli.GetVirtualMachine(ctx, vmID)
		if err != nil {
			return "", fmt.Errorf("get VM: %w", err)
		}
		if vm.Status == freeboxTypes.StoppedStatus || vm.Status == freeboxTypes.StoppingStatus {
			return freeboxTypes.StoppedStatus, nil
		}
		return vmStop(ctx, cli, vmID, killTimeout)
	default:
		return "", fmt.Errorf("unsupported VM status %q", desiredStatus)
	}
}

func ensureVMStoppedForConfigChange(ctx context.Context, cli client.Client, vmID int64, currentStatus string, killTimeout time.Duration) error {
	if currentStatus == freeboxTypes.StoppedStatus || currentStatus == freeboxTypes.StoppingStatus {
		return nil
	}
	if _, err := vmStop(ctx, cli, vmID, killTimeout); err != nil {
		return fmt.Errorf("stop VM before config change: %w", err)
	}
	return nil
}

func (VirtualMachine) Create(ctx context.Context, req infer.CreateRequest[VirtualMachineArgs]) (infer.CreateResponse[VirtualMachineState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VirtualMachineState]{}, err
	}

	timeouts := req.Inputs.Timeouts.resolved()
	status := desiredVMStatus(req.Inputs)

	if req.DryRun {
		return infer.CreateResponse[VirtualMachineState]{
			ID: "unknown",
			Output: VirtualMachineState{
				VirtualMachineArgs: req.Inputs,
				Status:             status,
				Timeouts:           req.Inputs.Timeouts,
			},
		}, nil
	}

	createCtx, cancel := context.WithTimeout(ctx, timeouts.Create)
	defer cancel()

	vm, err := cli.CreateVirtualMachine(createCtx, toPayload(req.Inputs))
	if err != nil {
		return infer.CreateResponse[VirtualMachineState]{}, fmt.Errorf("create VM: %w", err)
	}

	state := VirtualMachineState{
		VirtualMachineArgs: req.Inputs,
		ID:                 vm.ID,
		Mac:                vm.Mac,
		Status:             vm.Status,
		Timeouts:           req.Inputs.Timeouts,
	}

	newStatus, err := applyVMDesiredStatus(createCtx, cli, vm.ID, status, timeouts.Kill)
	if err != nil {
		return infer.CreateResponse[VirtualMachineState]{}, fmt.Errorf("set VM status: %w", err)
	}
	state.Status = newStatus

	fillVMNetworking(createCtx, cli, vm, &state, timeouts.Networking)

	return infer.CreateResponse[VirtualMachineState]{
		ID:     fmt.Sprintf("%d", vm.ID),
		Output: state,
	}, nil
}

func (VirtualMachine) Read(ctx context.Context, req infer.ReadRequest[VirtualMachineArgs, VirtualMachineState]) (infer.ReadResponse[VirtualMachineArgs, VirtualMachineState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[VirtualMachineArgs, VirtualMachineState]{}, err
	}

	timeouts := req.State.Timeouts.resolved()
	readCtx, cancel := context.WithTimeout(ctx, timeouts.Read)
	defer cancel()

	vm, err := cli.GetVirtualMachine(readCtx, req.State.ID)
	if err != nil {
		if errors.Is(err, client.ErrVirtualMachineNotFound) {
			return infer.ReadResponse[VirtualMachineArgs, VirtualMachineState]{}, nil
		}
		return infer.ReadResponse[VirtualMachineArgs, VirtualMachineState]{}, fmt.Errorf("get VM: %w", err)
	}

	state := VirtualMachineState{
		VirtualMachineArgs: req.State.VirtualMachineArgs,
		ID:                 vm.ID,
		Mac:                vm.Mac,
		Status:             vm.Status,
		Timeouts:           req.State.Timeouts,
	}
	fillVMNetworking(readCtx, cli, vm, &state, timeouts.Networking)
	return infer.ReadResponse[VirtualMachineArgs, VirtualMachineState]{State: state}, nil
}

func (VirtualMachine) Update(ctx context.Context, req infer.UpdateRequest[VirtualMachineArgs, VirtualMachineState]) (infer.UpdateResponse[VirtualMachineState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[VirtualMachineState]{}, err
	}

	timeouts := req.Inputs.Timeouts.resolved()
	desiredStatus := desiredVMStatus(req.Inputs)

	if req.DryRun {
		return infer.UpdateResponse[VirtualMachineState]{
			Output: VirtualMachineState{
				VirtualMachineArgs: req.Inputs,
				ID:                 req.State.ID,
				Mac:                req.State.Mac,
				Status:             desiredStatus,
				Timeouts:           req.Inputs.Timeouts,
			},
		}, nil
	}

	updateCtx, cancel := context.WithTimeout(ctx, timeouts.Update)
	defer cancel()

	if err := ensureVMStoppedForConfigChange(updateCtx, cli, req.State.ID, req.State.Status, timeouts.Kill); err != nil {
		return infer.UpdateResponse[VirtualMachineState]{}, err
	}

	vm, err := cli.UpdateVirtualMachine(updateCtx, req.State.ID, toPayload(req.Inputs))
	if err != nil {
		return infer.UpdateResponse[VirtualMachineState]{}, fmt.Errorf("update VM: %w", err)
	}

	newStatus, err := applyVMDesiredStatus(updateCtx, cli, vm.ID, desiredStatus, timeouts.Kill)
	if err != nil {
		return infer.UpdateResponse[VirtualMachineState]{}, fmt.Errorf("set VM status: %w", err)
	}

	state := VirtualMachineState{
		VirtualMachineArgs: req.Inputs,
		ID:                 vm.ID,
		Mac:                vm.Mac,
		Status:             newStatus,
		Timeouts:           req.Inputs.Timeouts,
	}

	fillVMNetworking(updateCtx, cli, vm, &state, timeouts.Networking)

	return infer.UpdateResponse[VirtualMachineState]{Output: state}, nil
}

func (VirtualMachine) Delete(ctx context.Context, req infer.DeleteRequest[VirtualMachineState]) (infer.DeleteResponse, error) {
	vmId := req.State.ID
	freeboxLog("[freebox] VirtualMachine Delete: vmId=%d\n", vmId)
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	timeouts := req.State.Timeouts.resolved()
	deleteCtx, cancel := context.WithTimeout(ctx, timeouts.Delete)
	defer cancel()

	if err := ensureVMStoppedForConfigChange(deleteCtx, cli, vmId, req.State.Status, timeouts.Kill); err != nil {
		freeboxLog("[freebox] VirtualMachine Delete vmId=%d: stop failed: %v\n", vmId, err)
		return infer.DeleteResponse{}, err
	}
	err = cli.DeleteVirtualMachine(deleteCtx, vmId)
	if err != nil {
		freeboxLog("[freebox] VirtualMachine Delete vmId=%d: %v\n", vmId, err)
		return infer.DeleteResponse{}, err
	}
	freeboxLog("[freebox] VirtualMachine Delete vmId=%d: success\n", vmId)
	return infer.DeleteResponse{}, nil
}

func vmStart(ctx context.Context, c client.Client, id int64) (string, error) {
	events, err := c.ListenEvents(ctx, []freeboxTypes.EventDescription{{Source: "vm", Name: "state_changed"}})
	if err != nil {
		return "", err
	}
	if err := c.StartVirtualMachine(ctx, id); err != nil {
		return "", err
	}
	for {
		select {
		case event := <-events:
			if event.Error != nil {
				return "", event.Error
			}
			var e vmStateEvent
			if err := json.Unmarshal(event.Notification.Result, &e); err != nil || e.ID != id {
				continue
			}
			if e.Status == freeboxTypes.RunningStatus {
				return e.Status, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func vmStop(ctx context.Context, c client.Client, id int64, killTimeout time.Duration) (string, error) {
	events, err := c.ListenEvents(ctx, []freeboxTypes.EventDescription{{Source: "vm", Name: "state_changed"}})
	if err != nil {
		return "", err
	}
	if err := c.StopVirtualMachine(ctx, id); err != nil {
		return "", err
	}
	deadline := time.After(killTimeout)
	for {
		select {
		case event := <-events:
			if event.Error != nil {
				return "", event.Error
			}
			var e vmStateEvent
			if err := json.Unmarshal(event.Notification.Result, &e); err != nil || e.ID != id {
				continue
			}
			if e.Status == freeboxTypes.StoppedStatus {
				return e.Status, nil
			}
		case <-deadline:
			if err := c.KillVirtualMachine(ctx, id); err != nil {
				freeboxLog("[freebox] VM vmId=%d: kill after stop timeout: %v\n", id, err)
			}
			return freeboxTypes.StoppedStatus, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// getNetworkBinds returns the LAN network binds (interface, IPv4, IPv6) for the VM by matching its MAC.
// It retries until the VM appears on the LAN or the timeout is reached.
func getNetworkBinds(ctx context.Context, c client.Client, vm freeboxTypes.VirtualMachine, timeout time.Duration) ([]NetworkingBind, error) {
	deadline := time.After(timeout)
	for {
		var binds []NetworkingBind
		interfaces, err := c.ListLanInterfaceInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("list LAN interfaces: %w", err)
		}
		for _, ifInfo := range interfaces {
			if ifInfo.HostCount == 0 {
				continue
			}
			hosts, err := c.GetLanInterface(ctx, ifInfo.Name)
			if err != nil {
				return nil, fmt.Errorf("get LAN interface %q: %w", ifInfo.Name, err)
			}
			for _, host := range hosts {
				if host.L2Ident.Type != "mac_address" || !strings.EqualFold(host.L2Ident.ID, vm.Mac) {
					continue
				}
				bind := NetworkingBind{Interface: ifInfo.Name}
				for _, conn := range host.L3Connectivities {
					if conn.Type == freeboxTypes.IPV4 {
						bind.Ipv4 = conn.Address
					}
					if conn.Type == freeboxTypes.IPV6 {
						bind.Ipv6 = append(bind.Ipv6, conn.Address)
					}
				}
				if bind.Ipv4 != "" {
					binds = append(binds, bind)
				}
			}
		}
		if len(binds) > 0 {
			return binds, nil
		}
		select {
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for VM to appear on LAN")
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(minDuration(timeout/10, 5*time.Second)):
		}
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
