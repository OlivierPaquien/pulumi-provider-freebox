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

// RemoteFile manages a file on the Freebox (download, copy, upload).
type RemoteFile struct{}

type RemoteFileArgs struct {
	DestinationPath  string                    `pulumi:"destinationPath"`
	SourceURL        *string                   `pulumi:"sourceUrl,optional"`
	SourceRemoteFile *string                   `pulumi:"sourceRemoteFile,optional"`
	SourceContent    *string                   `pulumi:"sourceContent,optional"`
	SourceLocalFile  *string                   `pulumi:"sourceLocalFile,optional"`
	Checksum         *string                   `pulumi:"checksum,optional"`
	Parents          *bool                     `pulumi:"parents,optional"`
	Authentication   *RemoteFileAuthentication `pulumi:"authentication,optional"`
	Extract          *RemoteFileExtract        `pulumi:"extract,optional"`
	Polling          *RemoteFilePolling        `pulumi:"polling,optional"`
}

type RemoteFileState struct {
	RemoteFileArgs
	Checksum   string `pulumi:"checksum"`
	SizeOnDisk int64  `pulumi:"sizeOnDisk"`
}

func (args *RemoteFileArgs) Annotate(a infer.Annotator) {
	a.Describe(&args.DestinationPath, "Path on the Freebox where the file will be stored.")
	a.Describe(&args.SourceURL, "URL of the file to download.")
	a.Describe(&args.SourceRemoteFile, "Path to an existing file on the Freebox to copy.")
	a.Describe(&args.SourceContent, "Inline file content to upload.")
	a.Describe(&args.SourceLocalFile, "Path to a local file to upload (relative to the Pulumi engine).")
	a.Describe(&args.Checksum, "Expected checksum (method:value). Computed after create if omitted.")
	a.Describe(&args.Parents, "Create parent directories if missing.")
}

func (st *RemoteFileState) Annotate(a infer.Annotator) {
	a.Describe(&st.Checksum, "Checksum of the file on disk.")
	a.Describe(&st.SizeOnDisk, "Size in bytes of the file on disk.")
}

func (RemoteFile) Annotate(a infer.Annotator) {
	a.SetToken("downloads", "File")
}

func (RemoteFile) Create(ctx context.Context, req infer.CreateRequest[RemoteFileArgs]) (infer.CreateResponse[RemoteFileState], error) {
	if err := validateRemoteFileArgs(req.Inputs); err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	if req.DryRun {
		return infer.CreateResponse[RemoteFileState]{
			ID:     req.Inputs.DestinationPath,
			Output: RemoteFileState{RemoteFileArgs: req.Inputs},
		}, nil
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	dest := req.Inputs.DestinationPath
	if _, err := cli.GetFileInfo(ctx, dest); err == nil {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("file already exists at %q; delete it or import the resource", dest)
	} else if !errors.Is(err, client.ErrPathNotFound) {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("check file: %w", err)
	}

	if err := createRemoteFile(ctx, cli, req.Inputs); err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	checksum, err := verifyRemoteFileChecksum(ctx, cli, req.Inputs, dest)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	if err := extractRemoteFile(ctx, cli, req.Inputs, dest); err != nil {
		return infer.CreateResponse[RemoteFileState]{}, err
	}

	info, err := cli.GetFileInfo(ctx, dest)
	if err != nil {
		return infer.CreateResponse[RemoteFileState]{}, fmt.Errorf("get file info: %w", err)
	}

	state := RemoteFileState{
		RemoteFileArgs: req.Inputs,
		Checksum:       checksum,
		SizeOnDisk:     int64(info.SizeBytes),
	}
	return infer.CreateResponse[RemoteFileState]{ID: dest, Output: state}, nil
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
		return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{}, err
	}

	state := req.State
	state.SizeOnDisk = int64(info.SizeBytes)
	return infer.ReadResponse[RemoteFileArgs, RemoteFileState]{State: state}, nil
}

func (RemoteFile) Update(ctx context.Context, req infer.UpdateRequest[RemoteFileArgs, RemoteFileState]) (infer.UpdateResponse[RemoteFileState], error) {
	old := req.State
	new := req.Inputs
	if err := validateRemoteFileArgs(new); err != nil {
		return infer.UpdateResponse[RemoteFileState]{}, err
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.UpdateResponse[RemoteFileState]{}, err
	}

	recreate := sourceChanged(old.RemoteFileArgs, new)
	if !recreate && old.Checksum != "" && ptrStr(new.Checksum) != "" && old.Checksum != ptrStr(new.Checksum) {
		recreate = true
	}

	polling := new.Polling.resolved()

	if recreate {
		deleteP := polling.Delete.withDefaults(defaultDeleteInterval(), defaultDeleteTimeout())
		if err := deleteFilesIfExist(ctx, cli, deleteP, old.DestinationPath); err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		if err := createRemoteFile(ctx, cli, new); err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		checksum, err := verifyRemoteFileChecksum(ctx, cli, new, new.DestinationPath)
		if err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		info, err := cli.GetFileInfo(ctx, new.DestinationPath)
		if err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		return infer.UpdateResponse[RemoteFileState]{Output: RemoteFileState{
			RemoteFileArgs: new,
			Checksum:       checksum,
			SizeOnDisk:     int64(info.SizeBytes),
		}}, nil
	}

	if old.DestinationPath != new.DestinationPath {
		if _, err := cli.GetFileInfo(ctx, new.DestinationPath); err == nil {
			return infer.UpdateResponse[RemoteFileState]{}, fmt.Errorf("file already exists at %q", new.DestinationPath)
		} else if !errors.Is(err, client.ErrPathNotFound) {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		task, err := cli.MoveFiles(ctx, []string{old.DestinationPath}, new.DestinationPath, freeboxTypes.FileMoveModeOverwrite)
		if err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		moveP := polling.Move.withDefaults(defaultDeleteInterval(), defaultDeleteTimeout())
		if err := waitFileSystemTask(ctx, cli, task.ID, moveP); err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		if err := stopAndDeleteFileSystemTask(ctx, cli, task.ID); err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		info, err := cli.GetFileInfo(ctx, new.DestinationPath)
		if err != nil {
			return infer.UpdateResponse[RemoteFileState]{}, err
		}
		state := RemoteFileState{
			RemoteFileArgs: new,
			Checksum:       old.Checksum,
			SizeOnDisk:     int64(info.SizeBytes),
		}
		return infer.UpdateResponse[RemoteFileState]{Output: state}, nil
	}

	return infer.UpdateResponse[RemoteFileState]{Output: old}, nil
}

func sourceChanged(old, new RemoteFileArgs) bool {
	return ptrStr(old.SourceURL) != ptrStr(new.SourceURL) ||
		ptrStr(old.SourceRemoteFile) != ptrStr(new.SourceRemoteFile) ||
		ptrStr(old.SourceContent) != ptrStr(new.SourceContent) ||
		ptrStr(old.SourceLocalFile) != ptrStr(new.SourceLocalFile)
}

func (RemoteFile) Delete(ctx context.Context, req infer.DeleteRequest[RemoteFileState]) (infer.DeleteResponse, error) {
	freeboxLog("[freebox] RemoteFile Delete: destinationPath=%q\n", req.State.DestinationPath)
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	polling := req.State.Polling.resolved()
	deleteP := polling.Delete.withDefaults(defaultDeleteInterval(), defaultDeleteTimeout())
	if err := deleteFilesIfExist(ctx, cli, deleteP, req.State.DestinationPath); err != nil {
		freeboxLog("[freebox] RemoteFile Delete path=%q: %v\n", req.State.DestinationPath, err)
		return infer.DeleteResponse{}, err
	}
	freeboxLog("[freebox] RemoteFile Delete path=%q: success\n", req.State.DestinationPath)
	return infer.DeleteResponse{}, nil
}

func defaultDeleteInterval() time.Duration { return time.Second }
func defaultDeleteTimeout() time.Duration { return time.Minute }
