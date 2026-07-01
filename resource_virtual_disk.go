package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VirtualDisk manages a virtual disk image on the Freebox.
type VirtualDisk struct{}

type VirtualDiskArgs struct {
	Path        string                `pulumi:"path"`
	Type        string                `pulumi:"type,optional"`
	VirtualSize int64                 `pulumi:"virtualSize"`
	ResizeFrom  *string               `pulumi:"resizeFrom,optional"`
	Polling     *VirtualDiskPolling   `pulumi:"polling,optional"`
}

type VirtualDiskState struct {
	VirtualDiskArgs
	Type               string `pulumi:"type"`
	SizeOnDisk         int64  `pulumi:"sizeOnDisk"`
	ResizeFromChecksum string `pulumi:"resizeFromChecksum,optional"`
}

func (args *VirtualDiskArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.Path, "Path to the virtual disk on the Freebox.")
	a.Describe(&args.Type, "Type of virtual disk (qcow2 or raw). Defaults to qcow2.")
	a.Describe(&args.VirtualSize, "Size in bytes of the virtual disk (as seen inside the VM).")
	a.Describe(&args.ResizeFrom, "Path to an existing disk to copy and resize from.")
}

func (st *VirtualDiskState) Annotate(a infer.Annotator) {
	a.Describe(&st.Type, "Type of virtual disk.")
	a.Describe(&st.SizeOnDisk, "Space in bytes used on disk.")
	a.Describe(&st.ResizeFromChecksum, "SHA512 checksum of resizeFrom source (internal).")
}

func (VirtualDisk) Annotate(a infer.Annotator) {
	a.SetToken("virtual", "Disk")
}

func (VirtualDisk) Create(ctx context.Context, req infer.CreateRequest[VirtualDiskArgs]) (infer.CreateResponse[VirtualDiskState], error) {
	if req.DryRun {
		return infer.CreateResponse[VirtualDiskState]{
			ID: req.Inputs.Path,
			Output: VirtualDiskState{
				VirtualDiskArgs: req.Inputs,
				Type:            virtualDiskType(req.Inputs),
			},
		}, nil
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, err
	}

	checksum, err := createVirtualDisk(ctx, cli, req.Inputs)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, err
	}

	state, err := virtualDiskStateFromAPI(ctx, cli, req.Inputs, checksum)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, err
	}
	return infer.CreateResponse[VirtualDiskState]{ID: req.Inputs.Path, Output: state}, nil
}

func (VirtualDisk) Read(ctx context.Context, req infer.ReadRequest[VirtualDiskArgs, VirtualDiskState]) (infer.ReadResponse[VirtualDiskArgs, VirtualDiskState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{}, err
	}

	info, err := cli.GetVirtualDiskInfo(ctx, req.State.Path)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == freeboxTypes.DiskErrorNotFound {
			return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{}, nil
		}
		fileInfo, fileErr := cli.GetFileInfo(ctx, req.State.Path)
		if fileErr != nil {
			return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{}, nil
		}
		state := req.State
		state.SizeOnDisk = int64(fileInfo.SizeBytes)
		return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{State: state}, nil
	}

	state := VirtualDiskState{
		VirtualDiskArgs:    req.State.VirtualDiskArgs,
		Type:               string(info.Type),
		SizeOnDisk:         info.ActualSize,
		ResizeFromChecksum: req.State.ResizeFromChecksum,
	}
	return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{State: state}, nil
}

func (VirtualDisk) Update(ctx context.Context, req infer.UpdateRequest[VirtualDiskArgs, VirtualDiskState]) (infer.UpdateResponse[VirtualDiskState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[VirtualDiskState]{}, err
	}

	state, err := updateVirtualDisk(ctx, cli, req.State.VirtualDiskArgs, req.Inputs, req.State.ResizeFromChecksum)
	if err != nil {
		return infer.UpdateResponse[VirtualDiskState]{}, err
	}
	return infer.UpdateResponse[VirtualDiskState]{Output: state}, nil
}

func (VirtualDisk) Delete(ctx context.Context, req infer.DeleteRequest[VirtualDiskState]) (infer.DeleteResponse, error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	polling := req.State.Polling.resolved()
	deleteP := polling.Delete.withDefaults(defaultDeleteInterval(), defaultDeleteTimeout())
	if err := deleteFilesIfExist(ctx, cli, deleteP, req.State.Path); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("delete virtual disk: %w", err)
	}
	return infer.DeleteResponse{}, nil
}
