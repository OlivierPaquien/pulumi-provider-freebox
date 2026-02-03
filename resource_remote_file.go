package main

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// RemoteFile resource: downloads a file from a URL to the Freebox.
// Simplified: sourceURL and destinationPath only (no checksum, extract, or auth).
type RemoteFile struct{}

type RemoteFileArgs struct {
	SourceURL       string `pulumi:"sourceUrl"`
	DestinationPath string `pulumi:"destinationPath"`
}

type RemoteFileState struct {
	RemoteFileArgs
	SizeOnDisk int64 `pulumi:"sizeOnDisk"`
}

func (RemoteFile) Annotate(a infer.Annotator) {
	a.Describe(&RemoteFile{}, "Downloads a file from a URL and stores it on the Freebox.")
	args := &RemoteFileArgs{}
	a.Describe(&args.SourceURL, "URL of the file to download.")
	a.Describe(&args.DestinationPath, "Path on the Freebox where the file will be stored.")
	st := &RemoteFileState{}
	a.Describe(&st.SizeOnDisk, "Size in bytes of the file on disk.")
}

func (RemoteFile) Create(ctx context.Context, req infer.CreateRequest[RemoteFileArgs]) (infer.CreateResponse[RemoteFileState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	dest := req.Inputs.DestinationPath
	dir := path.Dir(dest)
	filename := path.Base(dest)

	if req.DryRun {
		return infer.CreateResponse[RemoteFileState]{
			ID:     dest,
			Output: RemoteFileState{RemoteFileArgs: req.Inputs},
		}, nil
	}

	_, err = cli.GetFileInfo(ctx, dest)
	if err == nil {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("file already exists at %s", dest)
	}
	if err != nil && !errors.Is(err, client.ErrPathNotFound) {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("check file: %w", err)
	}

	payload := freeboxTypes.DownloadRequest{
		DownloadURLs:      []string{req.Inputs.SourceURL},
		DownloadDirectory: dir,
		Filename:          filename,
	}

	taskID, err := cli.AddDownloadTask(ctx, payload)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("add download task: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := waitDownloadTask(waitCtx, cli, taskID); err != nil {
		_ = stopAndDeleteDownloadTask(ctx, cli, taskID)
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("wait download: %w", err)
	}
	_ = stopAndDeleteDownloadTask(ctx, cli, taskID)

	info, err := cli.GetFileInfo(ctx, dest)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("get file info: %w", err)
	}

	state := RemoteFileState{
		RemoteFileArgs: req.Inputs,
		SizeOnDisk:     int64(info.SizeBytes),
	}
	return infer.CreateResponse[RemoteFileState]{
		ID:     dest,
		Output: state,
	}, nil
}

func (RemoteFile) Read(ctx context.Context, req infer.ReadRequest[RemoteFileArgs, RemoteFileState]) (infer.ReadResponse[RemoteFileArgs, RemoteFileState], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{}, err
	}

	info, err := cli.GetFileInfo(ctx, req.State.DestinationPath)
	if err != nil {
		if errors.Is(err, client.ErrPathNotFound) {
			return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{}, nil
		}
		return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{}, fmt.Errorf("get file info: %w", err)
	}

	state := RemoteFileState{
		RemoteFileArgs: req.State.RemoteFileArgs,
		SizeOnDisk:     int64(info.SizeBytes),
	}
	return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{State: state}, nil
}

func (RemoteFile) Delete(ctx context.Context, req infer.DeleteRequest[RemoteFileState]) error {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return err
	}
	task, err := cli.RemoveFiles(ctx, []string{req.State.DestinationPath})
	if err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_ = waitFileSystemTask(waitCtx, cli, task.ID)
	_ = stopAndDeleteFileSystemTask(ctx, cli, task.ID)
	return nil
}

func waitDownloadTask(ctx context.Context, c client.Client, taskID int64) error {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
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
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func stopAndDeleteDownloadTask(ctx context.Context, c client.Client, taskID int64) error {
	_ = c.UpdateDownloadTask(ctx, taskID, freeboxTypes.DownloadTaskUpdate{
		Status: freeboxTypes.DownloadTaskStatusStopped,
	})
	if err := c.DeleteDownloadTask(ctx, taskID); err != nil && !errors.Is(err, client.ErrTaskNotFound) {
		return err
	}
	return nil
}
