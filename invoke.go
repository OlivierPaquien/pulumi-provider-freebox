package main

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetApiVersion invokes the Freebox API version discovery.
type GetApiVersion struct{}

type GetApiVersionArgs struct{}

type GetApiVersionResult struct {
	UID            string `pulumi:"uid"`
	APIVersion     string `pulumi:"apiVersion"`
	APIDomain      string `pulumi:"apiDomain"`
	APIBaseURL     string `pulumi:"apiBaseUrl"`
	BoxModelName   string `pulumi:"boxModelName"`
	BoxModel       string `pulumi:"boxModel"`
	HTTPSPort      int64  `pulumi:"httpsPort"`
	HTTPSAvailable bool   `pulumi:"httpsAvailable"`
}

func (GetApiVersion) Annotate(a infer.Annotator) {
	a.Describe(&GetApiVersion{}, "Discovery of the Freebox over HTTP(S).")
	r := &GetApiVersionResult{}
	a.Describe(&r.UID, "Device unique id.")
	a.Describe(&r.APIVersion, "Current API version on the Freebox.")
	a.Describe(&r.APIDomain, "Domain to use in place of hardcoded Freebox IP.")
	a.Describe(&r.APIBaseURL, "API root path on the HTTP server.")
	a.Describe(&r.BoxModel, "Box model.")
	a.Describe(&r.BoxModelName, "Box model display name.")
	a.Describe(&r.HTTPSPort, "Port for remote HTTPS access.")
	a.Describe(&r.HTTPSAvailable, "Whether HTTPS is configured.")
}

func (GetApiVersion) Invoke(ctx context.Context, req infer.FunctionRequest[GetApiVersionArgs]) (infer.FunctionResponse[GetApiVersionResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetApiVersionResult]{}, err
	}

	info, err := cli.APIVersion(ctx)
	if err != nil {
		return infer.FunctionResponse[GetApiVersionResult]{}, fmt.Errorf("get API version: %w", err)
	}

	return infer.FunctionResponse[GetApiVersionResult]{
		Output: GetApiVersionResult{
			UID:            info.UID,
			APIVersion:     info.APIVersion,
			APIDomain:      info.APIDomain,
			APIBaseURL:     info.APIBaseURL,
			BoxModelName:   info.BoxModelName,
			BoxModel:       info.BoxModel,
			HTTPSPort:      int64(info.HTTPSPort),
			HTTPSAvailable: info.HTTPSAvailable,
		},
	}, nil
}

// GetVirtualDisk invokes virtual disk info by path.
type GetVirtualDisk struct{}

type GetVirtualDiskArgs struct {
	Path string `pulumi:"path"`
}

type GetVirtualDiskResult struct {
	Path        string `pulumi:"path"`
	Type        string `pulumi:"type"`
	ActualSize  int64  `pulumi:"actualSize"`
	VirtualSize int64  `pulumi:"virtualSize"`
}

func (GetVirtualDisk) Annotate(a infer.Annotator) {
	a.Describe(&GetVirtualDisk{}, "Get information about a virtual disk.")
	args := &GetVirtualDiskArgs{}
	a.Describe(&args.Path, "Path to the virtual disk.")
	res := &GetVirtualDiskResult{}
	a.Describe(&res.Type, "Type of virtual disk.")
	a.Describe(&res.ActualSize, "Space in bytes used on disk.")
	a.Describe(&res.VirtualSize, "Size in bytes as seen inside the VM.")
}

func (GetVirtualDisk) Invoke(ctx context.Context, req infer.FunctionRequest[GetVirtualDiskArgs]) (infer.FunctionResponse[GetVirtualDiskResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVirtualDiskResult]{}, err
	}

	info, err := cli.GetVirtualDiskInfo(ctx, req.Input.Path)
	if err != nil {
		return infer.FunctionResponse[GetVirtualDiskResult]{}, fmt.Errorf("get virtual disk info: %w", err)
	}

	return infer.FunctionResponse[GetVirtualDiskResult]{
		Output: GetVirtualDiskResult{
			Path:        req.Input.Path,
			Type:        string(info.Type),
			ActualSize:  info.ActualSize,
			VirtualSize: info.VirtualSize,
		},
	}, nil
}
