package main

import (
	"context"
	"errors"
	"fmt"
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

func waitFileSystemTask(ctx context.Context, c client.Client, taskID int64) error {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		task, err := c.GetFileSystemTask(ctx, taskID)
		if err != nil {
			if errors.Is(err, client.ErrTaskNotFound) {
				return nil
			}
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
			return ctx.Err()
		case <-tick.C:
		}
	}
}
