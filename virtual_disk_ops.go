package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func virtualDiskType(args VirtualDiskArgs) string {
	if args.Type != "" {
		return args.Type
	}
	return freeboxTypes.QCow2Disk
}

func createVirtualDiskFromScratch(ctx context.Context, c client.Client, args VirtualDiskArgs) error {
	polling := args.Polling.resolved()
	taskID, err := c.CreateVirtualDisk(ctx, freeboxTypes.VirtualDisksCreatePayload{
		DiskPath: freeboxTypes.Base64Path(args.Path),
		Size:     args.VirtualSize,
		DiskType: virtualDiskType(args),
	})
	if err != nil {
		return err
	}
	p := polling.Create.withDefaults(time.Second, time.Minute)
	if err := waitVirtualDiskTaskWithTimeout(ctx, c, taskID, p.Timeout); err != nil {
		return err
	}
	return stopAndDeleteVirtualDiskTask(ctx, c, taskID)
}

func createVirtualDiskFromResizeFrom(ctx context.Context, c client.Client, args VirtualDiskArgs) (checksum string, err error) {
	resizeFrom := *args.ResizeFrom
	polling := args.Polling.resolved()

	hashTask, err := c.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashTypeSHA512,
		Path:     freeboxTypes.Base64Path(resizeFrom),
	})
	if err != nil {
		return "", err
	}

	copyTask, err := c.CopyFiles(ctx, []string{resizeFrom}, args.Path, freeboxTypes.FileCopyModeOverwrite)
	if err != nil {
		return "", err
	}
	copyP := polling.Copy.withDefaults(2*time.Second, 2*time.Minute)
	if err := waitFileSystemTask(ctx, c, copyTask.ID, copyP); err != nil {
		return "", err
	}
	if err := stopAndDeleteFileSystemTask(ctx, c, copyTask.ID); err != nil {
		return "", err
	}

	checksumP := polling.Checksum.withDefaults(time.Second, time.Minute)
	if err := waitFileSystemTask(ctx, c, hashTask.ID, checksumP); err != nil {
		return "", err
	}
	hash, err := c.GetHashResult(ctx, hashTask.ID)
	if err != nil {
		return "", err
	}
	if err := stopAndDeleteFileSystemTask(ctx, c, hashTask.ID); err != nil {
		return "", err
	}

	resizeTaskID, err := c.ResizeVirtualDisk(ctx, freeboxTypes.VirtualDisksResizePayload{
		DiskPath:    freeboxTypes.Base64Path(args.Path),
		NewSize:     args.VirtualSize - 1024,
		ShrinkAllow: true,
	})
	if err != nil {
		return "", err
	}
	resizeP := polling.Resize.withDefaults(time.Second, time.Minute)
	if err := waitVirtualDiskTaskWithTimeout(ctx, c, resizeTaskID, resizeP.Timeout); err != nil {
		return "", err
	}
	if err := stopAndDeleteVirtualDiskTask(ctx, c, resizeTaskID); err != nil {
		return "", err
	}

	hashJSON, err := json.Marshal(hash)
	if err != nil {
		return "", err
	}
	return string(hashJSON), nil
}

func createVirtualDisk(ctx context.Context, c client.Client, args VirtualDiskArgs) (resizeFromChecksum string, err error) {
	if args.ResizeFrom != nil && *args.ResizeFrom != "" {
		return createVirtualDiskFromResizeFrom(ctx, c, args)
	}
	return "", createVirtualDiskFromScratch(ctx, c, args)
}

func virtualDiskNeedsRecreate(old, new VirtualDiskArgs, oldChecksum string) (bool, error) {
	if virtualDiskType(old) != virtualDiskType(new) {
		return true, nil
	}
	if oldChecksum == "" || new.ResizeFrom == nil || *new.ResizeFrom == "" {
		return false, nil
	}
	// Compare SHA512 of new resize_from with stored checksum when resize_from unchanged path-wise
	if old.ResizeFrom != nil && new.ResizeFrom != nil && *old.ResizeFrom == *new.ResizeFrom {
		return false, nil
	}
	return true, nil
}

func updateVirtualDisk(ctx context.Context, c client.Client, old, new VirtualDiskArgs, oldChecksum string) (VirtualDiskState, error) {
	polling := new.Polling.resolved()

	recreate, err := virtualDiskNeedsRecreate(old, new, oldChecksum)
	if err != nil {
		return VirtualDiskState{}, err
	}

	if !recreate && oldChecksum != "" && new.ResizeFrom != nil && *new.ResizeFrom != "" {
		if old.ResizeFrom == nil || *old.ResizeFrom != *new.ResizeFrom {
			var stored string
			if err := json.Unmarshal([]byte(oldChecksum), &stored); err == nil {
				p := polling.Checksum.withDefaults(time.Second, time.Minute)
				hash, hashErr := computeFileChecksum(ctx, c, *new.ResizeFrom, string(freeboxTypes.HashTypeSHA512), p)
				if hashErr != nil {
					return VirtualDiskState{}, hashErr
				}
				recreate = stored != hash
			}
		}
	}

	if recreate {
		deleteP := polling.Delete.withDefaults(time.Second, time.Minute)
		if err := deleteFilesIfExist(ctx, c, deleteP, old.Path); err != nil {
			return VirtualDiskState{}, err
		}
		checksum, err := createVirtualDisk(ctx, c, new)
		if err != nil {
			return VirtualDiskState{}, err
		}
		return virtualDiskStateFromAPI(ctx, c, new, checksum)
	}

	if old.Path != new.Path {
		task, err := c.MoveFiles(ctx, []string{old.Path}, new.Path, freeboxTypes.FileMoveModeSkip)
		if err != nil {
			return VirtualDiskState{}, err
		}
		moveP := polling.Move.withDefaults(time.Second, time.Minute)
		if err := waitFileSystemTask(ctx, c, task.ID, moveP); err != nil {
			return VirtualDiskState{}, err
		}
		if err := stopAndDeleteFileSystemTask(ctx, c, task.ID); err != nil {
			return VirtualDiskState{}, err
		}
	}

	if old.VirtualSize != new.VirtualSize {
		taskID, err := c.ResizeVirtualDisk(ctx, freeboxTypes.VirtualDisksResizePayload{
			DiskPath:    freeboxTypes.Base64Path(new.Path),
			NewSize:     new.VirtualSize - 1024,
			ShrinkAllow: true,
		})
		if err != nil {
			return VirtualDiskState{}, err
		}
		resizeP := polling.Resize.withDefaults(time.Second, time.Minute)
		if err := waitVirtualDiskTaskWithTimeout(ctx, c, taskID, resizeP.Timeout); err != nil {
			return VirtualDiskState{}, err
		}
		if err := stopAndDeleteVirtualDiskTask(ctx, c, taskID); err != nil {
			return VirtualDiskState{}, err
		}
	}

	return virtualDiskStateFromAPI(ctx, c, new, oldChecksum)
}

func virtualDiskStateFromAPI(ctx context.Context, c client.Client, args VirtualDiskArgs, resizeFromChecksum string) (VirtualDiskState, error) {
	info, err := c.GetVirtualDiskInfo(ctx, args.Path)
	if err != nil {
		return VirtualDiskState{}, fmt.Errorf("get virtual disk info: %w", err)
	}
	return VirtualDiskState{
		VirtualDiskArgs:    args,
		Type:               string(info.Type),
		SizeOnDisk:         info.ActualSize,
		ResizeFromChecksum: resizeFromChecksum,
	}, nil
}
