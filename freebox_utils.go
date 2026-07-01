package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func stopAndDeleteFileSystemTask(ctx context.Context, c client.Client, taskID int64) error {
	_, _ = c.UpdateFileSystemTask(ctx, taskID, freeboxTypes.FileSytemTaskUpdate{
		State: freeboxTypes.FileTaskStatePaused,
	})
	if err := c.DeleteFileSystemTask(ctx, taskID); err != nil && !errors.Is(err, client.ErrTaskNotFound) {
		return fmt.Errorf("delete file system task: %w", err)
	}
	return nil
}

func stopAndDeleteVirtualDiskTask(ctx context.Context, c client.Client, taskID int64) error {
	if err := c.DeleteVirtualDiskTask(ctx, taskID); err != nil && !errors.Is(err, client.ErrTaskNotFound) {
		return fmt.Errorf("delete virtual disk task: %w", err)
	}
	return nil
}

func stopAndDeleteUploadTask(ctx context.Context, c client.Client, taskID int64) error {
	_ = c.CancelUploadTask(ctx, taskID)
	if err := c.DeleteUploadTask(ctx, taskID); err != nil && !errors.Is(err, client.ErrTaskNotFound) {
		return fmt.Errorf("delete upload task: %w", err)
	}
	return nil
}

func stopAndDeleteDownloadTask(ctx context.Context, c client.Client, taskID int64) error {
	_ = c.UpdateDownloadTask(ctx, taskID, freeboxTypes.DownloadTaskUpdate{
		Status: freeboxTypes.DownloadTaskStatusStopped,
	})
	if err := c.DeleteDownloadTask(ctx, taskID); err != nil && !errors.Is(err, client.ErrTaskNotFound) {
		return fmt.Errorf("delete download task: %w", err)
	}
	return nil
}

func waitFileSystemTask(ctx context.Context, c client.Client, taskID int64, polling PollingSpec) error {
	interval := polling.Interval
	if interval == 0 {
		interval = time.Second
	}
	timeout := polling.Timeout
	if timeout == 0 {
		timeout = time.Minute
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		task, err := c.GetFileSystemTask(ctx, taskID)
		if err != nil {
			return err
		}
		switch task.State {
		case freeboxTypes.FileTaskStateDone:
			return nil
		case freeboxTypes.FileTaskStateFailed:
			return fmt.Errorf("file system task failed: %s", task.Error)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("file system task timed out: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

func waitDownloadTaskWithPolling(ctx context.Context, c client.Client, taskID int64, polling PollingSpec) error {
	interval := polling.Interval
	if interval == 0 {
		interval = 3 * time.Second
	}
	timeout := polling.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		task, err := c.GetDownloadTask(ctx, taskID)
		if err != nil {
			return err
		}
		switch task.Status {
		case freeboxTypes.DownloadTaskStatusDone:
			return nil
		case freeboxTypes.DownloadTaskStatusError:
			return fmt.Errorf("download failed: %s", task.Error)
		case freeboxTypes.DownloadTaskStatusStopped:
			return fmt.Errorf("download stopped")
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("download timed out: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

func waitUploadTask(ctx context.Context, c client.Client, taskID int64, polling PollingSpec) error {
	interval := polling.Interval
	if interval == 0 {
		interval = 3 * time.Second
	}
	timeout := polling.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		task, err := c.GetUploadTask(ctx, taskID)
		if err != nil {
			return err
		}
		switch task.Status {
		case freeboxTypes.UploadTaskStatusDone:
			return nil
		case freeboxTypes.UploadTaskStatusFailed,
			freeboxTypes.UploadTaskStatusCancelled,
			freeboxTypes.UploadTaskStatusConflict,
			freeboxTypes.UploadTaskStatusTimeout:
			return fmt.Errorf("upload failed: %s", task.Status)
		default:
			if task.Uploaded == task.Size {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("upload timed out: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

func waitVirtualDiskTaskWithTimeout(ctx context.Context, c client.Client, taskID int64, timeout time.Duration) error {
	if timeout == 0 {
		timeout = time.Minute
	}

	events, err := c.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: freeboxTypes.EventSourceVMDisk,
		Name:   freeboxTypes.EventDiskTaskDone,
	}})
	if err != nil {
		return err
	}

	task, err := c.GetVirtualDiskTask(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Done {
		if task.Error {
			return fmt.Errorf("virtual disk task failed")
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case event := <-events:
			if !event.Notification.Success {
				return fmt.Errorf("virtual disk task failed: %v", event.Error)
			}
			return nil
		case <-ctx.Done():
			return fmt.Errorf("virtual disk task timed out: %w", ctx.Err())
		}
	}
}

func hashSpec(checksum string) (method, value string) {
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) == 1 {
		return string(freeboxTypes.HashTypeSHA256), parts[0]
	}
	if parts[0] == "" {
		parts[0] = string(freeboxTypes.HashTypeSHA256)
	}
	return parts[0], parts[1]
}

func computeFileChecksum(ctx context.Context, c client.Client, path, hashType string, polling PollingSpec) (string, error) {
	if hashType == "" {
		hashType = string(freeboxTypes.HashTypeSHA256)
	}
	task, err := c.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashType(hashType),
		Path:     freeboxTypes.Base64Path(path),
	})
	if err != nil {
		return "", err
	}
	if err := waitFileSystemTask(ctx, c, task.ID, polling); err != nil {
		return "", err
	}
	result, err := c.GetHashResult(ctx, task.ID)
	if err != nil {
		return "", err
	}
	_ = stopAndDeleteFileSystemTask(ctx, c, task.ID)
	return result, nil
}

func int64Ptr(i int64) *int64 { return &i }

func boolPtr(v bool) *bool { return &v }

func deleteFilesIfExist(ctx context.Context, c client.Client, polling PollingSpec, paths ...string) error {
	var toDelete []string
	for _, p := range paths {
		if _, err := c.GetFileInfo(ctx, p); err == nil {
			toDelete = append(toDelete, p)
		} else if !errors.Is(err, client.ErrPathNotFound) {
			return err
		}
	}
	if len(toDelete) == 0 {
		return nil
	}
	task, err := c.RemoveFiles(ctx, toDelete)
	if err != nil {
		return err
	}
	if err := waitFileSystemTask(ctx, c, task.ID, polling); err != nil {
		return err
	}
	return stopAndDeleteFileSystemTask(ctx, c, task.ID)
}
