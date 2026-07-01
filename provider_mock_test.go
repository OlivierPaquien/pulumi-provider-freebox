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

func TestProvider_PortForwardingCreateMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const (
		ruleID   = int64(42)
		wanPort  = int64(2222)
		targetIP = "192.168.1.10"
	)
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/fw/redir/", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, mockPortForwardingRuleJSON(ruleID, wanPort, targetIP))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), "mock-pf")
	props := property.NewMap(map[string]property.Value{
		"enabled":        property.New(true),
		"ipProtocol":     property.New("tcp"),
		"portRangeStart": property.New(float64(wanPort)),
		"targetIp":       property.New(targetIP),
		"comment":        property.New("pulumi-test"),
	})

	resp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
	require.NoError(t, err)
	assert.Equal(t, "42", resp.ID)
	assert.Equal(t, property.New(float64(ruleID)), resp.Properties.Get("ruleId"))
	assert.Equal(t, property.New(targetIP), resp.Properties.Get("targetIp"))
}

func TestProvider_PortForwardingReadMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const (
		ruleID   = int64(5)
		wanPort  = int64(8080)
		targetIP = "192.168.1.20"
	)
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/fw/redir/%d", mockAPIVersion, ruleID), r.URL.Path)
		writeJSON(w, http.StatusOK, mockPortForwardingRuleJSON(ruleID, wanPort, targetIP))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), "mock-pf-read")
	state := property.NewMap(map[string]property.Value{
		"enabled":        property.New(true),
		"ipProtocol":     property.New("tcp"),
		"portRangeStart": property.New(float64(wanPort)),
		"targetIp":       property.New(targetIP),
		"ruleId":         property.New(float64(ruleID)),
	})

	resp, err := server.Read(p.ReadRequest{
		ID:         "5",
		Urn:        urn,
		Properties: state,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Properties)
	assert.Equal(t, property.New(targetIP), resp.Properties.Get("targetIp"))
	assert.Equal(t, property.New(float64(wanPort)), resp.Properties.Get("portRangeStart"))
}

func TestProvider_PortForwardingReadNotFoundMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{
			"success": false,
			"error_code": "noent",
			"msg": "not found"
		}`)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), "mock-pf-missing")

	_, err := server.Read(p.ReadRequest{
		ID:  "99",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"ruleId": property.New(float64(99)),
		}),
	})
	require.NoError(t, err, "read should succeed when rule is absent (refresh semantics)")
}

func TestProvider_PortForwardingReadServerErrorMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), "mock-pf-error")

	_, err := server.Read(p.ReadRequest{
		ID:  "5",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"ruleId": property.New(float64(5)),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get port forwarding")
}

func TestProvider_PortForwardingDeleteMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	const ruleID = int64(7)
	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/fw/redir/%d", mockAPIVersion, ruleID), r.URL.Path)
		writeJSON(w, http.StatusOK, `{"success": true}`)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(portForwardingType), "mock-pf-del")
	props := property.NewMap(map[string]property.Value{
		"ruleId":         property.New(float64(ruleID)),
		"enabled":        property.New(true),
		"ipProtocol":     property.New("tcp"),
		"portRangeStart": property.New(float64(22)),
		"targetIp":       property.New("192.168.1.10"),
		"hostname":       property.New(""),
	})

	err := server.Delete(p.DeleteRequest{ID: "7", Urn: urn, Properties: props})
	require.NoError(t, err)
}

func TestProvider_VirtualMachineReadNotFoundMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, fmt.Sprintf("/api/%s/vm/1234", mockAPIVersion), r.URL.Path)
		writeJSON(w, http.StatusOK, `{
			"success": false,
			"error_code": "no_such_vm",
			"msg": "vm not found"
		}`)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachineType), "mock-vm-missing")

	_, err := server.Read(p.ReadRequest{
		ID:  "1234",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":     property.New(float64(1234)),
			"name":     property.New("test-vm"),
			"diskPath": property.New("Freebox/VMs/test.qcow2"),
			"diskType": property.New("qcow2"),
			"memory":   property.New(float64(256)),
			"vcpus":    property.New(float64(1)),
			"status":   property.New("stopped"),
		}),
	})
	require.NoError(t, err, "read should succeed when VM is absent (refresh semantics)")
}

func TestProvider_VirtualMachineReadServerErrorMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachineType), "mock-vm-error")

	_, err := server.Read(p.ReadRequest{
		ID:  "1234",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":     property.New(float64(1234)),
			"name":     property.New("test-vm"),
			"diskPath": property.New("Freebox/VMs/test.qcow2"),
			"diskType": property.New("qcow2"),
			"memory":   property.New(float64(256)),
			"vcpus":    property.New(float64(1)),
		}),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get VM")
}

func TestProvider_VirtualMachineReadSuccessMock(t *testing.T) {
	srv, seq := newSequentialMockServer(t)
	defer srv.Close()

	seq.append(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, mockVirtualMachineJSON(1234, "test-vm", "stopped"))
	})

	server := newTestServer(t, mockTestConfig(srv.URL))
	urn := resource.NewURN("stack", "proj", "", tokens.Type(virtualMachineType), "mock-vm-read")

	resp, err := server.Read(p.ReadRequest{
		ID:  "1234",
		Urn: urn,
		Properties: property.NewMap(map[string]property.Value{
			"vmId":     property.New(float64(1234)),
			"name":     property.New("test-vm"),
			"diskPath": property.New("Freebox/VMs/test.qcow2"),
			"diskType": property.New("qcow2"),
			"memory":   property.New(float64(256)),
			"vcpus":    property.New(float64(1)),
			"status":   property.New("stopped"),
		}),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Properties)
	assert.Equal(t, property.New("stopped"), resp.Properties.Get("status"))
	assert.Equal(t, property.New("f6:69:9c:d9:4f:3d"), resp.Properties.Get("mac"))
}
