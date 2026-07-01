package main

import (
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

// LanHostL2Ident is the L2 identity of a LAN host.
type LanHostL2Ident struct {
	L2ID string `pulumi:"l2Id"` // id is reserved by Pulumi
	Type string `pulumi:"type"`
}

// LanHostL3Connectivity is a layer-3 connectivity entry for a LAN host.
type LanHostL3Connectivity struct {
	Address   string `pulumi:"address"`
	Active    bool   `pulumi:"active"`
	Reachable bool   `pulumi:"reachable"`
	Type      string `pulumi:"type"`
}

// LanHostName is a host name and its source.
type LanHostName struct {
	Name   string `pulumi:"name"`
	Source string `pulumi:"source"`
}

// LanHostNetworkControl summarizes network control profile for a host.
type LanHostNetworkControl struct {
	ProfileID   int64  `pulumi:"profileId"`
	Name        string `pulumi:"name"`
	CurrentMode string `pulumi:"currentMode"`
}

// LanHost is the full LAN host object returned by port forwarding rules.
type LanHost struct {
	HostID               string                  `pulumi:"hostId"` // id is reserved by Pulumi
	Active               bool                    `pulumi:"active"`
	Reachable            bool                    `pulumi:"reachable"`
	Persistent           bool                    `pulumi:"persistent"`
	PrimaryNameManual    bool                    `pulumi:"primaryNameManual"`
	VendorName           string                  `pulumi:"vendorName"`
	HostType             string                  `pulumi:"hostType"`
	Interface            string                  `pulumi:"interface"`
	FirstActivitySeconds *float64                `pulumi:"firstActivitySeconds,optional"`
	PrimaryName          string                  `pulumi:"primaryName"`
	DefaultName          string                  `pulumi:"defaultName"`
	L2Ident              LanHostL2Ident          `pulumi:"l2ident"`
	L3Connectivities     []LanHostL3Connectivity `pulumi:"l3connectivities"`
	Names                []LanHostName           `pulumi:"names"`
	NetworkControl       *LanHostNetworkControl  `pulumi:"networkControl,optional"`
}

func lanHostFromAPI(host freeboxTypes.LanInterfaceHost) LanHost {
	l3 := make([]LanHostL3Connectivity, len(host.L3Connectivities))
	for i, c := range host.L3Connectivities {
		l3[i] = LanHostL3Connectivity{
			Address:   c.Address,
			Active:    c.Active,
			Reachable: c.Reachable,
			Type:      string(c.Type),
		}
	}
	names := make([]LanHostName, len(host.Names))
	for i, n := range host.Names {
		names[i] = LanHostName{Name: n.Name, Source: n.Source}
	}
	out := LanHost{
		HostID:            host.ID,
		Active:            host.Active,
		Reachable:         host.Reachable,
		Persistent:        host.Persistent,
		PrimaryNameManual: host.PrimaryNameManual,
		VendorName:        host.VendorName,
		HostType:          string(host.Type),
		Interface:         host.Interface,
		PrimaryName:       host.PrimaryName,
		DefaultName:       host.DefaultName,
		L2Ident: LanHostL2Ident{
			L2ID: host.L2Ident.ID,
			Type: string(host.L2Ident.Type),
		},
		L3Connectivities: l3,
		Names:            names,
	}
	if !host.FirstActivity.IsZero() {
		v := float64(host.FirstActivity.UnixMicro()) / 1_000_000.0
		out.FirstActivitySeconds = &v
	}
	if host.NetworkControl != nil {
		out.NetworkControl = &LanHostNetworkControl{
			ProfileID:   int64(host.NetworkControl.ProfileID),
			Name:        host.NetworkControl.Name,
			CurrentMode: host.NetworkControl.CurrentMode,
		}
	}
	return out
}

func lanHostPtrFromAPI(host *freeboxTypes.LanInterfaceHost) *LanHost {
	if host == nil {
		return nil
	}
	h := lanHostFromAPI(*host)
	return &h
}

// LanInterfaceHostRef references a host on a LAN interface.
type LanInterfaceHostRef struct {
	Interface string         `pulumi:"interface"`
	HostID    string         `pulumi:"hostId"`
	L2Ident   LanHostL2Ident `pulumi:"l2ident"`
}

// DhcpLeaseInfo describes a DHCP static lease.
type DhcpLeaseInfo struct {
	LeaseID  string `pulumi:"leaseId"` // id is reserved by Pulumi
	Mac      string `pulumi:"mac"`
	IP       string `pulumi:"ip"`
	Hostname string `pulumi:"hostname"`
	Comment  string `pulumi:"comment"`
}

func dhcpLeaseFromAPI(lease freeboxTypes.DHCPStaticLeaseInfo) DhcpLeaseInfo {
	return DhcpLeaseInfo{
		LeaseID:  lease.ID,
		Mac:      normalizeMAC(lease.Mac),
		IP:       lease.IP,
		Hostname: lease.Hostname,
		Comment:  lease.Comment,
	}
}

func lanHostRefFromAPI(host freeboxTypes.LanInterfaceHost) LanInterfaceHostRef {
	return LanInterfaceHostRef{
		Interface: host.Interface,
		HostID:    host.ID,
		L2Ident: LanHostL2Ident{
			L2ID: host.L2Ident.ID,
			Type: string(host.L2Ident.Type),
		},
	}
}
