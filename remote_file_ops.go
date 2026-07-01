package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

type RemoteFileAuthentication struct {
	BasicAuth *RemoteFileBasicAuth `pulumi:"basicAuth,optional"`
}

type RemoteFileBasicAuth struct {
	Username string `pulumi:"username,optional"`
	Password string `pulumi:"password,optional"`
}

type RemoteFileExtract struct {
	DestinationPath string `pulumi:"destinationPath,optional"`
	Password        string `pulumi:"password,optional"`
	Overwrite       *bool  `pulumi:"overwrite,optional"`
}

func remoteFileSourceCount(args RemoteFileArgs) int {
	n := 0
	if args.SourceURL != nil && *args.SourceURL != "" {
		n++
	}
	if args.SourceRemoteFile != nil && *args.SourceRemoteFile != "" {
		n++
	}
	if args.SourceContent != nil && *args.SourceContent != "" {
		n++
	}
	if args.SourceLocalFile != nil && *args.SourceLocalFile != "" {
		n++
	}
	return n
}

func validateRemoteFileArgs(args RemoteFileArgs) error {
	if args.DestinationPath == "" {
		return fmt.Errorf("destinationPath is required")
	}
	switch n := remoteFileSourceCount(args); n {
	case 0:
		return fmt.Errorf("exactly one of sourceUrl, sourceRemoteFile, sourceContent, or sourceLocalFile must be set")
	case 1:
		return nil
	default:
		return fmt.Errorf("only one of sourceUrl, sourceRemoteFile, sourceContent, or sourceLocalFile may be set")
	}
}

func createRemoteFileParents(ctx context.Context, c client.Client, dest string) error {
	parent := "/"
	for _, segment := range strings.Split(strings.TrimPrefix(path.Dir(dest), "/"), "/") {
		if segment == "" {
			continue
		}
		newParent, err := c.CreateDirectory(ctx, parent, segment)
		if err != nil {
			if errors.Is(err, client.ErrDestinationConflict) {
				info, infoErr := c.GetFileInfo(ctx, path.Join(parent, segment))
				if infoErr != nil {
					return infoErr
				}
				if info.Type == freeboxTypes.FileTypeDirectory {
					parent = path.Join(parent, segment)
					continue
				}
			}
			return fmt.Errorf("create parent directory %s: %w", path.Join(parent, segment), err)
		}
		parent = newParent
	}
	return nil
}

func downloadPayload(args RemoteFileArgs) freeboxTypes.DownloadRequest {
	dest := args.DestinationPath
	payload := freeboxTypes.DownloadRequest{
		DownloadURLs:      []string{*args.SourceURL},
		DownloadDirectory: path.Dir(dest),
		Filename:          path.Base(dest),
	}
	if args.Checksum != nil {
		payload.Hash = *args.Checksum
	}
	if args.Authentication != nil && args.Authentication.BasicAuth != nil {
		auth := args.Authentication.BasicAuth
		if auth.Username != "" {
			payload.Username = auth.Username
		}
		if auth.Password != "" {
			payload.Password = auth.Password
		}
	}
	return payload
}

func createRemoteFileFromURL(ctx context.Context, c client.Client, args RemoteFileArgs, polling RemoteFilePolling) error {
	taskID, err := c.AddDownloadTask(ctx, downloadPayload(args))
	if err != nil {
		return err
	}
	p := polling.Download.withDefaults(3*time.Second, 30*time.Minute)
	if err := waitDownloadTaskWithPolling(ctx, c, taskID, p); err != nil {
		_ = stopAndDeleteDownloadTask(ctx, c, taskID)
		return err
	}
	return stopAndDeleteDownloadTask(ctx, c, taskID)
}

func createRemoteFileFromRemote(ctx context.Context, c client.Client, args RemoteFileArgs, polling RemoteFilePolling) error {
	task, err := c.CopyFiles(ctx, []string{*args.SourceRemoteFile}, args.DestinationPath, freeboxTypes.FileCopyModeOverwrite)
	if err != nil {
		return err
	}
	p := polling.Copy.withDefaults(time.Second, time.Minute)
	if err := waitFileSystemTask(ctx, c, task.ID, p); err != nil {
		return err
	}
	return stopAndDeleteFileSystemTask(ctx, c, task.ID)
}

func createRemoteFileFromContent(ctx context.Context, c client.Client, args RemoteFileArgs, polling RemoteFilePolling) error {
	p := polling.Upload.withDefaults(3*time.Second, 30*time.Minute)
	content := *args.SourceContent
	dest := args.DestinationPath

	uploadCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	writer, taskID, err := c.FileUploadStart(uploadCtx, freeboxTypes.FileUploadStartActionInput{
		Size:     len(content),
		Dirname:  freeboxTypes.Base64Path(path.Dir(dest)),
		Filename: path.Base(dest),
		Force:    freeboxTypes.FileUploadStartActionForceOverwrite,
	})
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		return err
	}
	if err := waitUploadTask(uploadCtx, c, taskID, p); err != nil {
		_ = stopAndDeleteUploadTask(ctx, c, taskID)
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return stopAndDeleteUploadTask(ctx, c, taskID)
}

func createRemoteFileFromLocalFile(ctx context.Context, c client.Client, args RemoteFileArgs, polling RemoteFilePolling) error {
	localPath := *args.SourceLocalFile
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}

	p := polling.Upload.withDefaults(3*time.Second, 30*time.Minute)
	uploadCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	dest := args.DestinationPath
	writer, taskID, err := c.FileUploadStart(uploadCtx, freeboxTypes.FileUploadStartActionInput{
		Size:     int(info.Size()),
		Dirname:  freeboxTypes.Base64Path(path.Dir(dest)),
		Filename: path.Base(dest),
		Force:    freeboxTypes.FileUploadStartActionForceOverwrite,
	})
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, f); err != nil {
		_ = stopAndDeleteUploadTask(ctx, c, taskID)
		return err
	}
	if err := waitUploadTask(uploadCtx, c, taskID, p); err != nil {
		_ = stopAndDeleteUploadTask(ctx, c, taskID)
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return stopAndDeleteUploadTask(ctx, c, taskID)
}

func createRemoteFile(ctx context.Context, c client.Client, args RemoteFileArgs) error {
	if err := validateRemoteFileArgs(args); err != nil {
		return err
	}
	polling := args.Polling.resolved()
	if args.Parents != nil && *args.Parents {
		if err := createRemoteFileParents(ctx, c, args.DestinationPath); err != nil {
			return err
		}
	}
	switch {
	case args.SourceURL != nil && *args.SourceURL != "":
		return createRemoteFileFromURL(ctx, c, args, polling)
	case args.SourceRemoteFile != nil && *args.SourceRemoteFile != "":
		return createRemoteFileFromRemote(ctx, c, args, polling)
	case args.SourceContent != nil && *args.SourceContent != "":
		return createRemoteFileFromContent(ctx, c, args, polling)
	default:
		return createRemoteFileFromLocalFile(ctx, c, args, polling)
	}
}

func verifyRemoteFileChecksum(ctx context.Context, c client.Client, args RemoteFileArgs, dest string) (string, error) {
	polling := args.Polling.resolved()
	hMethod, expected := hashSpec(ptrStr(args.Checksum))
	p := polling.ChecksumCompute.withDefaults(time.Second, 2*time.Minute)
	result, err := computeFileChecksum(ctx, c, dest, hMethod, p)
	if err != nil {
		return "", err
	}
	if expected == "" {
		return fmt.Sprintf("%s:%s", hMethod, result), nil
	}
	if expected != result {
		return "", fmt.Errorf("checksum mismatch: expected %q, got %q", expected, result)
	}
	if args.Checksum != nil && *args.Checksum != "" {
		return *args.Checksum, nil
	}
	return fmt.Sprintf("%s:%s", hMethod, result), nil
}

func extractRemoteFile(ctx context.Context, c client.Client, args RemoteFileArgs, src string) error {
	if args.Extract == nil || args.Extract.DestinationPath == "" {
		return nil
	}
	overwrite := false
	if args.Extract.Overwrite != nil {
		overwrite = *args.Extract.Overwrite
	}
	task, err := c.ExtractFile(ctx, freeboxTypes.ExtractFilePayload{
		Src:       freeboxTypes.Base64Path(src),
		Dst:       freeboxTypes.Base64Path(args.Extract.DestinationPath),
		Password:  args.Extract.Password,
		Overwrite: overwrite,
	})
	if err != nil {
		return err
	}
	p := args.Polling.resolved().Extract.withDefaults(time.Second, 2*time.Minute)
	if err := waitFileSystemTask(ctx, c, task.ID, p); err != nil {
		return err
	}
	return stopAndDeleteFileSystemTask(ctx, c, task.ID)
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
