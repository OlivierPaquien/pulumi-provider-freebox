package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikolalohinski/free-go/client"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// GetDhcpLease looks up a DHCP static lease by MAC address.
type GetDhcpLease struct{}

type GetDhcpLeaseArgs struct {
	Mac string `pulumi:"mac"`
}

type GetDhcpLeaseResult struct {
	DhcpLeaseInfo
}

func (GetDhcpLease) Annotate(a infer.Annotator) {
	a.SetToken("dhcp", "getLease")
}

func (GetDhcpLease) Invoke(ctx context.Context, req infer.FunctionRequest[GetDhcpLeaseArgs]) (infer.FunctionResponse[GetDhcpLeaseResult], error) {
	if !isValidMAC(req.Input.Mac) {
		return infer.FunctionResponse[GetDhcpLeaseResult]{}, fmt.Errorf("mac must be a valid MAC address")
	}

	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetDhcpLeaseResult]{}, err
	}

	lease, err := cli.GetDHCPStaticLease(ctx, normalizeMAC(req.Input.Mac))
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "noent" {
			return infer.FunctionResponse[GetDhcpLeaseResult]{}, fmt.Errorf("no DHCP static lease found for MAC %q", req.Input.Mac)
		}
		return infer.FunctionResponse[GetDhcpLeaseResult]{}, fmt.Errorf("get DHCP lease: %w", err)
	}

	return infer.FunctionResponse[GetDhcpLeaseResult]{
		Output: GetDhcpLeaseResult{DhcpLeaseInfo: dhcpLeaseFromAPI(lease)},
	}, nil
}

// GetDhcpLeases lists all DHCP static leases.
type GetDhcpLeases struct{}

type GetDhcpLeasesArgs struct{}

type GetDhcpLeasesResult struct {
	Leases []DhcpLeaseInfo `pulumi:"leases"`
}

func (GetDhcpLeases) Annotate(a infer.Annotator) {
	a.SetToken("dhcp", "getLeases")
}

func (GetDhcpLeases) Invoke(ctx context.Context, _ infer.FunctionRequest[GetDhcpLeasesArgs]) (infer.FunctionResponse[GetDhcpLeasesResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetDhcpLeasesResult]{}, err
	}

	leases, err := cli.ListDHCPStaticLease(ctx)
	if err != nil {
		return infer.FunctionResponse[GetDhcpLeasesResult]{}, fmt.Errorf("list DHCP leases: %w", err)
	}

	out := make([]DhcpLeaseInfo, len(leases))
	for i, lease := range leases {
		out[i] = dhcpLeaseFromAPI(lease)
	}

	return infer.FunctionResponse[GetDhcpLeasesResult]{
		Output: GetDhcpLeasesResult{Leases: out},
	}, nil
}

// GetLanConfig reads the LAN configuration.
type GetLanConfig struct{}

type GetLanConfigArgs struct{}

type GetLanConfigResult struct {
	IP          string `pulumi:"ip"`
	Name        string `pulumi:"name"`
	NameDNS     string `pulumi:"nameDns"`
	NameMDNS    string `pulumi:"nameMdns"`
	NameNetBIOS string `pulumi:"nameNetbios"`
	Mode        string `pulumi:"mode"`
}

func (GetLanConfig) Annotate(a infer.Annotator) {
	a.SetToken("lan", "getConfig")
}

func (GetLanConfig) Invoke(ctx context.Context, _ infer.FunctionRequest[GetLanConfigArgs]) (infer.FunctionResponse[GetLanConfigResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanConfigResult]{}, err
	}

	config, err := cli.GetLanConfig(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanConfigResult]{}, fmt.Errorf("get LAN config: %w", err)
	}

	return infer.FunctionResponse[GetLanConfigResult]{
		Output: GetLanConfigResult{
			IP:          config.IP,
			Name:        config.Name,
			NameDNS:     config.NameDNS,
			NameMDNS:    config.NameMDNS,
			NameNetBIOS: config.NameNetBIOS,
			Mode:        config.Mode,
		},
	}, nil
}

// GetLanInterfaces lists LAN interfaces.
type GetLanInterfaces struct{}

type GetLanInterfacesArgs struct{}

type LanInterfaceInfo struct {
	Name      string `pulumi:"name"`
	HostCount int64  `pulumi:"hostCount"`
}

type GetLanInterfacesResult struct {
	Interfaces []LanInterfaceInfo `pulumi:"interfaces"`
}

func (GetLanInterfaces) Annotate(a infer.Annotator) {
	a.SetToken("lan", "getInterfaces")
}

func (GetLanInterfaces) Invoke(ctx context.Context, _ infer.FunctionRequest[GetLanInterfacesArgs]) (infer.FunctionResponse[GetLanInterfacesResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfacesResult]{}, err
	}

	infos, err := cli.ListLanInterfaceInfo(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfacesResult]{}, fmt.Errorf("list LAN interfaces: %w", err)
	}

	out := make([]LanInterfaceInfo, len(infos))
	for i, info := range infos {
		out[i] = LanInterfaceInfo{Name: info.Name, HostCount: int64(info.HostCount)}
	}

	return infer.FunctionResponse[GetLanInterfacesResult]{
		Output: GetLanInterfacesResult{Interfaces: out},
	}, nil
}

// GetLanInterfaceHosts lists hosts on a LAN interface.
type GetLanInterfaceHosts struct{}

type GetLanInterfaceHostsArgs struct {
	Interface string `pulumi:"interface"`
}

type GetLanInterfaceHostsResult struct {
	Hosts []LanInterfaceHostRef `pulumi:"hosts"`
}

func (GetLanInterfaceHosts) Annotate(a infer.Annotator) {
	a.SetToken("lan", "getInterfaceHosts")
}

func (GetLanInterfaceHosts) Invoke(ctx context.Context, req infer.FunctionRequest[GetLanInterfaceHostsArgs]) (infer.FunctionResponse[GetLanInterfaceHostsResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfaceHostsResult]{}, err
	}

	hosts, err := cli.GetLanInterface(ctx, req.Input.Interface)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfaceHostsResult]{}, fmt.Errorf("get LAN interface hosts: %w", err)
	}

	out := make([]LanInterfaceHostRef, len(hosts))
	for i, host := range hosts {
		out[i] = lanHostRefFromAPI(host)
	}

	return infer.FunctionResponse[GetLanInterfaceHostsResult]{
		Output: GetLanInterfaceHostsResult{Hosts: out},
	}, nil
}

// GetLanInterfaceHost reads a single host on a LAN interface.
type GetLanInterfaceHost struct{}

type GetLanInterfaceHostArgs struct {
	Interface string `pulumi:"interface"`
	HostID    string `pulumi:"hostId"`
}

type GetLanInterfaceHostResult struct {
	LanInterfaceHostRef
}

func (GetLanInterfaceHost) Annotate(a infer.Annotator) {
	a.SetToken("lan", "getInterfaceHost")
}

func (GetLanInterfaceHost) Invoke(ctx context.Context, req infer.FunctionRequest[GetLanInterfaceHostArgs]) (infer.FunctionResponse[GetLanInterfaceHostResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfaceHostResult]{}, err
	}

	host, err := cli.GetLanInterfaceHost(ctx, req.Input.Interface, req.Input.HostID)
	if err != nil {
		return infer.FunctionResponse[GetLanInterfaceHostResult]{}, fmt.Errorf("get LAN interface host: %w", err)
	}

	return infer.FunctionResponse[GetLanInterfaceHostResult]{
		Output: GetLanInterfaceHostResult{LanInterfaceHostRef: lanHostRefFromAPI(host)},
	}, nil
}

// GetVmDistributions lists VM OS distributions available on the Freebox.
type GetVmDistributions struct{}

type GetVmDistributionsArgs struct{}

type VmDistribution struct {
	Hash string `pulumi:"hash"`
	OS   string `pulumi:"os"`
	URL  string `pulumi:"url"`
	Name string `pulumi:"name"`
}

type GetVmDistributionsResult struct {
	Distributions []VmDistribution `pulumi:"distributions"`
}

func (GetVmDistributions) Annotate(a infer.Annotator) {
	a.SetToken("virtual", "getDistributions")
}

func (GetVmDistributions) Invoke(ctx context.Context, _ infer.FunctionRequest[GetVmDistributionsArgs]) (infer.FunctionResponse[GetVmDistributionsResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVmDistributionsResult]{}, err
	}

	dists, err := cli.GetVirtualMachineDistributions(ctx)
	if err != nil {
		return infer.FunctionResponse[GetVmDistributionsResult]{}, fmt.Errorf("list VM distributions: %w", err)
	}

	out := make([]VmDistribution, len(dists))
	for i, d := range dists {
		out[i] = VmDistribution{Hash: d.Hash, OS: d.OS, URL: d.URL, Name: d.Name}
	}

	return infer.FunctionResponse[GetVmDistributionsResult]{
		Output: GetVmDistributionsResult{Distributions: out},
	}, nil
}

// GetSystemInfo reads Freebox system information.
type GetSystemInfo struct{}

type GetSystemInfoArgs struct{}

type GetSystemInfoResult struct {
	FirmwareVersion  string `pulumi:"firmwareVersion"`
	Mac              string `pulumi:"mac"`
	Serial           string `pulumi:"serial"`
	Uptime           string `pulumi:"uptime"`
	UptimeVal        int64  `pulumi:"uptimeVal"`
	BoardName        string `pulumi:"boardName"`
	TempCPUM         int64  `pulumi:"tempCpum"`
	TempSW           int64  `pulumi:"tempSw"`
	TempCPUB         int64  `pulumi:"tempCpub"`
	FanRPM           int64  `pulumi:"fanRpm"`
	BoxAuthenticated bool   `pulumi:"boxAuthenticated"`
	UserMainStorage  string `pulumi:"userMainStorage"`
}

func (GetSystemInfo) Annotate(a infer.Annotator) {
	a.SetToken("system", "getInfo")
}

func (GetSystemInfo) Invoke(ctx context.Context, _ infer.FunctionRequest[GetSystemInfoArgs]) (infer.FunctionResponse[GetSystemInfoResult], error) {
	cli, err := getFreeboxClient(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSystemInfoResult]{}, err
	}

	info, err := cli.GetSystemInfo(ctx)
	if err != nil {
		return infer.FunctionResponse[GetSystemInfoResult]{}, fmt.Errorf("get system info: %w", err)
	}

	return infer.FunctionResponse[GetSystemInfoResult]{
		Output: GetSystemInfoResult{
			FirmwareVersion:  info.FirmwareVersion,
			Mac:              info.Mac,
			Serial:           info.Serial,
			Uptime:           info.Uptime,
			UptimeVal:        info.UptimeVal,
			BoardName:        info.BoardName,
			TempCPUM:         int64(info.TempCPUM),
			TempSW:           int64(info.TempSW),
			TempCPUB:         int64(info.TempCPUB),
			FanRPM:           int64(info.FanRPM),
			BoxAuthenticated: info.BoxAuthenticated,
			UserMainStorage:  info.UserMainStorage,
		},
	}, nil
}
