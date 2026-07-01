package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

const previewMAC = "02:00:00:00:00:00"

func normalizeMAC(mac string) string {
	return strings.ToUpper(mac)
}

// macForPreview returns a syntactically valid MAC when the real value is not
// yet known (pulumi preview / dry-run).
func macForPreview(mac string) string {
	if isValidMAC(mac) {
		return normalizeMAC(mac)
	}
	return previewMAC
}

// dhcpLeaseHostname preserves the configured hostname when the API returns the MAC as placeholder.
func dhcpLeaseHostname(lease freeboxTypes.DHCPStaticLeaseInfo, configuredHostname string) string {
	hostname := lease.Hostname
	if configuredHostname == "" {
		return hostname
	}
	if strings.EqualFold(normalizeMAC(hostname), normalizeMAC(lease.Mac)) {
		return configuredHostname
	}
	return hostname
}

func dhcpLeaseStateFromAPI(lease freeboxTypes.DHCPStaticLeaseInfo, configuredHostname, configuredMAC string) DHCPStaticLeaseArgs {
	mac := normalizeMAC(lease.Mac)
	if configuredMAC != "" && strings.EqualFold(configuredMAC, lease.Mac) {
		mac = configuredMAC
	}
	return DHCPStaticLeaseArgs{
		Mac:      mac,
		IP:       lease.IP,
		Comment:  lease.Comment,
		Hostname: dhcpLeaseHostname(lease, configuredHostname),
	}
}

func isValidIPv4(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() != nil
}

func isValidMAC(mac string) bool {
	_, err := net.ParseMAC(strings.TrimSpace(mac))
	return err == nil
}

func validateDHCPStaticLeaseArgs(mac, ip string) error {
	if !isValidMAC(mac) {
		return fmt.Errorf("mac must be a valid MAC address")
	}
	if !isValidIPv4(ip) {
		return fmt.Errorf("ip must be a valid IPv4 address")
	}
	return nil
}

func ensureDHCPStaticLease(ctx context.Context, c client.Client, payload freeboxTypes.DHCPStaticLeasePayload) error {
	payload.Mac = normalizeMAC(payload.Mac)
	lease, err := c.GetDHCPStaticLease(ctx, payload.Mac)
	if err != nil {
		if _, createErr := c.CreateDHCPStaticLease(ctx, payload); createErr != nil {
			return fmt.Errorf("failed to create DHCP static lease for %s: %w", payload.Mac, createErr)
		}
		return nil
	}
	if strings.EqualFold(lease.IP, payload.IP) && lease.Comment == payload.Comment && lease.Hostname == payload.Hostname {
		return nil
	}
	if err := c.DeleteDHCPStaticLease(ctx, payload.Mac); err != nil {
		return fmt.Errorf("failed to replace DHCP static lease for %s: %w", payload.Mac, err)
	}
	if _, err := c.CreateDHCPStaticLease(ctx, payload); err != nil {
		return fmt.Errorf("failed to recreate DHCP static lease for %s: %w", payload.Mac, err)
	}
	return nil
}

func deleteDHCPStaticLeaseIfExists(ctx context.Context, c client.Client, mac string) error {
	mac = normalizeMAC(mac)
	if _, err := c.GetDHCPStaticLease(ctx, mac); err != nil {
		return nil
	}
	if err := c.DeleteDHCPStaticLease(ctx, mac); err != nil {
		return fmt.Errorf("failed to delete DHCP static lease for %s: %w", mac, err)
	}
	return nil
}
