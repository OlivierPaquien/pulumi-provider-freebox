package main

import (
	"fmt"
	"net/http"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	virtualMachinePowerType = "freebox:virtual:MachinePower"
	vpnServerType           = "freebox:vpn:Server"
	vpnUserType             = "freebox:vpn:User"
)

func TestProvider_VirtualMachinePowerCreateAlreadyStoppedMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const vmID = int64(1234)
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vm/%d", mockAPIVersion, vmID), r.URL.Path)
		writeJSON(w, http.StatusOK, mockVirtualMachineJSON(vmID, "test-vm", "stopped"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachinePowerType), "mock-power")

	resp, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":       property.New(float64(vmID)),
			"powerState": property.New("stopped"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "1234", resp.ID)
}

func TestProvider_VirtualMachinePowerReadMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const vmID = int64(1234)
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVirtualMachineJSON(vmID, "test-vm", "running"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachinePowerType), "mock-power-read")

	resp, err := server.Read(p.ReadRequest{
		ID:  "1234",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":       property.New(float64(vmID)),
			"powerState": property.New("stopped"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, property.New("running"), resp.Properties.Get("powerState"))
}

func TestProvider_VirtualMachinePowerDeleteAlreadyStoppedMock(t *testing.T) {
	server := newTestServer(t, mockTestConfig("http://127.0.0.1:1")) // no HTTP expected
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachinePowerType), "mock-power-del")

	err := server.Delete(p.DeleteRequest{
		ID:  "1234",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":       property.New(float64(1234)),
			"powerState": property.New("stopped"),
		}),
	})
	require.NoError(t, err)
}

func TestProvider_VirtualMachinePowerInvalidPowerStateMock(t *testing.T) {
	server := newTestServer(t, mockTestConfig("http://127.0.0.1:1"))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachinePowerType), "mock-power-bad")

	_, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":       property.New(float64(1)),
			"powerState": property.New("paused"),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "powerState")
}

func TestProvider_VpnServerCreateMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(false, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnServerType), "mock-vpn-server")

	resp, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"enabled": property.New(true),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, vpnServerID, resp.ID)
	assert.True(t, resp.Properties.Get("enabled").AsBool())
	assert.Equal(t, property.New(float64(1194)), resp.Properties.Get("serverPort"))
}

func TestProvider_VpnServerReadMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnServerType), "mock-vpn-read")

	resp, err := server.Read(p.ReadRequest{
		ID:         vpnServerID,
		Urn:        urn,
		Properties: property.NewMap(map[string]property.Value{}),
	})
	require.NoError(t, err)
	assert.True(t, resp.Properties.Get("enabled").AsBool())
	assert.Contains(t, resp.Properties.Get("ca").AsString(), "BEGIN CERTIFICATE")
}

func TestProvider_VpnServerDeletePreservesConfigMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	var putSeen bool
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		putSeen = true
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(false, 1194))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnServerType), "mock-vpn-del")

	err := server.Delete(p.DeleteRequest{
		ID:  vpnServerID,
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"enabled":    property.New(true),
			"serverPort": property.New(float64(1194)),
			"serverIp":   property.New("10.8.0.0"),
			"ca":         property.New("-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----\n"),
		}),
	})
	require.NoError(t, err)
	assert.True(t, putSeen)
}

func TestProvider_VpnUserCreateMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const login = "alice"
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/user/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockVPNUserJSON(login, "test user"))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/user/%s/config/openvpn", mockAPIVersion, login), r.URL.Path)
		writeJSON(w, http.StatusOK, mockVPNClientConfigJSON("client\nproto udp\n"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user")

	resp, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"login":       property.New(login),
			"password":    property.New("secret12"),
			"description": property.New("test user"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, login, resp.ID)
	assert.Contains(t, resp.Properties.Get("ovpnConfig").AsString(), "proto udp")
}

func TestProvider_VpnUserReadMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const login = "bob"
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVPNUserJSON(login, "desc"))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVPNClientConfigJSON("client\nremote vpn.example.com\n"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user-read")

	resp, err := server.Read(p.ReadRequest{
		ID:  login,
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"login":    property.New(login),
			"password": property.New("keep-me1"),
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, property.New(login), resp.Properties.Get("login"))
	assert.Equal(t, property.New("keep-me1"), resp.Properties.Get("password"))
	assert.Contains(t, resp.Properties.Get("ovpnConfig").AsString(), "vpn.example.com")
}

func TestProvider_VpnUserReadNotFoundMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVPNUserNotFoundJSON())
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user-missing")

	_, err := server.Read(p.ReadRequest{
		ID:  "ghost",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"login": property.New("ghost"),
		}),
	})
	require.NoError(t, err, "read should succeed when VPN user is absent (refresh semantics)")
}

func TestProvider_VpnUserDeleteMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const login = "alice"
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/user/%s", mockAPIVersion, login), r.URL.Path)
		writeJSON(w, http.StatusOK, `{"success": true}`)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user-del")

	err := server.Delete(p.DeleteRequest{
		ID:  login,
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"login":      property.New(login),
			"password":   property.New("secret12"),
			"ovpnConfig": property.New("client\n"),
		}),
	})
	require.NoError(t, err)
}

func TestProvider_VpnUserUpdateMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const login = "alice"
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		writeJSON(w, http.StatusOK, mockVPNUserJSON(login, "updated"))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockOpenVPNServerJSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVPNClientConfigJSON("client\nremote updated.example.com\n"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user-up")

	resp, err := server.Update(p.UpdateRequest{
		ID:  login,
		Urn: urn,
		State: property.NewMap(map[string]property.Value{
			"login":      property.New(login),
			"password":   property.New("oldpass1"),
			"ovpnConfig": property.New("old-config"),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"login":       property.New(login),
			"password":    property.New("newsecret"),
			"description": property.New("updated"),
		}),
		OldInputs: property.NewMap(map[string]property.Value{
			"login":    property.New(login),
			"password": property.New("oldpass1"),
		}),
	})
	require.NoError(t, err)
	assert.Contains(t, resp.Properties.Get("ovpnConfig").AsString(), "updated.example.com")
}

func TestProvider_VpnServerReadMockV4Fallback(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockAPIError404JSON())
	})
	for _, handler := range mockLoginHandlers() {
		seq.append(handler)
	}
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/user/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, `{"success": true, "result": []}`)
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn_routed/config/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerV4JSON(true, 1194))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnServerType), "mock-vpn-read-v4")

	resp, err := server.Read(p.ReadRequest{
		ID:         vpnServerID,
		Urn:        urn,
		Properties: property.NewMap(map[string]property.Value{}),
	})
	require.NoError(t, err)
	assert.True(t, resp.Properties.Get("enabled").AsBool())
	assert.Equal(t, property.New(float64(1194)), resp.Properties.Get("serverPort"))
}

func TestProvider_VpnUserCreateMockV4Fallback(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const login = "alice"
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockAPIError404JSON())
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		writeJSON(w, http.StatusOK, mockVPNUserJSON(login, "test user"))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockAPIError404JSON())
	})
	for _, handler := range mockLoginHandlers() {
		seq.append(handler)
	}
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/user/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, `{"success": true, "result": []}`)
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/openvpn_routed/config/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockOpenVPNServerV4JSON(true, 1194))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockVPNServerListV4JSON("openvpn_routed", "started"))
	})
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, fmt.Sprintf("/api/%s/vpn/download_config/openvpn_routed/%s", mockAPIVersion, login), r.URL.Path)
		w.Header().Set("Content-Type", "application/x-openvpn-profile")
		_, _ = w.Write([]byte("client\nproto udp\nremote vpn.example.com\n"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(vpnUserType), "mock-vpn-user-v4")

	resp, err := server.Create(p.CreateRequest{
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"login":       property.New(login),
			"password":    property.New("secret12"),
			"description": property.New("test user"),
		}),
	})
	require.NoError(t, err)
	assert.Contains(t, resp.Properties.Get("ovpnConfig").AsString(), "proto udp")
}
