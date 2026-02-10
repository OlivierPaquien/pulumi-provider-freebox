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
	// GetApiVersion est une struct vide ; l'Annotator reçoit ce type, pas GetApiVersionResult.
	// Tout appel à Describe (sur le type ou sur des champs du résultat) déclenche un panic
	// (reflect.Value.Addr of unaddressable value). Ne pas appeler Describe.
	a.SetToken("api", "Version")
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
	// GetVirtualDisk est une struct vide ; l'Annotator reçoit ce type. Ne pas appeler Describe
	// pour éviter le panic "reflect.Value.Addr of unaddressable value".
	a.SetToken("virtual", "getVirtualDisk")
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
