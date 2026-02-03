package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// VirtualDisk resource: manages a virtual disk image on the Freebox.
// This is a simplified implementation: create from scratch only (no resize_from).
type VirtualDisk struct{}

type VirtualDiskArgs struct {
	Path        string `pulumi:"path"`
	Type        string `pulumi:"type,optional"`
	VirtualSize int64  `pulumi:"virtualSize"`
}

type VirtualDiskState struct {
	VirtualDiskArgs
	Type       string `pulumi:"type"`
	SizeOnDisk int64  `pulumi:"sizeOnDisk"`
}

func (VirtualDisk) Annotate(a infer.Annotator) {
	a.Describe(&VirtualDisk{}, "Manages a virtual disk image within a Freebox.")
	args := &VirtualDiskArgs{}
	a.Describe(&args.Path, "Path to the virtual disk on the Freebox.")
	a.Describe(&args.Type, "Type of virtual disk (qcow2 or raw). Defaults to qcow2.")
	a.Describe(&args.VirtualSize, "Size in bytes of the virtual disk (as seen inside the VM).")
	st := &VirtualDiskState{}
	a.Describe(&st.SizeOnDisk, "Space in bytes used on disk.")
}

func (VirtualDisk) Create(ctx context.Context, req infer.CreateRequest[VirtualDiskArgs]) (infer.CreateResponse[VirtualDiskState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, err
	}

	diskType := req.Inputs.Type
	if diskType == "" {
		diskType = freeboxTypes.QCow2Disk
	}

	payload := freeboxTypes.VirtualDisksCreatePayload{
		DiskPath: freeboxTypes.Base64Path(req.Inputs.Path),
		Size:     req.Inputs.VirtualSize,
		DiskType: diskType,
	}

	if req.DryRun {
		return infer.CreateResponse[VirtualDiskState]{
			ID: req.Inputs.Path,
			Output: VirtualDiskState{
				VirtualDiskArgs: req.Inputs,
				Type:            diskType,
			},
		}, nil
	}

	taskID, err := cli.CreateVirtualDisk(ctx, payload)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, fmt.Errorf("create virtual disk: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := waitVirtualDiskTask(waitCtx, cli, taskID); err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, fmt.Errorf("wait for virtual disk creation: %w", err)
	}
	_ = cli.DeleteVirtualDiskTask(ctx, taskID)

	info, err := cli.GetVirtualDiskInfo(ctx, req.Inputs.Path)
	if err != nil {
		return infer.CreateResponse[VirtualDiskState]{}, fmt.Errorf("get virtual disk info: %w", err)
	}

	state := VirtualDiskState{
		VirtualDiskArgs: req.Inputs,
		Type:            string(info.Type),
		SizeOnDisk:      info.ActualSize,
	}
	return infer.CreateResponse[VirtualDiskState]{
		ID:     req.Inputs.Path,
		Output: state,
	}, nil
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
		VirtualDiskArgs: req.State.VirtualDiskArgs,
		Type:            string(info.Type),
		SizeOnDisk:      info.ActualSize,
	}
	return infer.ReadResponse[VirtualDiskArgs, VirtualDiskState]{State: state}, nil
}

func (VirtualDisk) Delete(ctx context.Context, req infer.DeleteRequest[VirtualDiskState]) error {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return err
	}
	task, err := cli.RemoveFiles(ctx, []string{req.State.Path})
	if err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_ = waitFileSystemTask(waitCtx, cli, task.ID)
	_ = stopAndDeleteFileSystemTask(ctx, cli, task.ID)
	return nil
}

func waitVirtualDiskTask(ctx context.Context, c client.Client, taskID int64) error {
	events, err := c.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: freeboxTypes.EventSourceVMDisk,
		Name:   freeboxTypes.EventDiskTaskDone,
	}})
	if err != nil {
		return err
	}
	for {
		task, err := c.GetVirtualDiskTask(ctx, taskID)
		if err != nil {
			return err
		}
		if task.Done {
			if task.Error {
				return fmt.Errorf("disk task failed")
			}
			return nil
		}
		select {
		case event := <-events:
			if !event.Notification.Success {
				return fmt.Errorf("disk task failed: %v", event.Error)
			}
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
