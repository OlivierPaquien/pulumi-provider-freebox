// Integration tests for freebox:virtual:MachinePower.
// Require FREEBOX_TOKEN and either FREEBOX_TEST_VM_ID or FREEBOX_TEST_DISK_PATH.
package main

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func virtualMachinePowerURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(virtualMachinePowerType), name)
}

func testVirtualMachinePower(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	var (
		vmID       int64
		cleanupVM  func()
		ownsVM     bool
	)
	if idStr := os.Getenv("FREEBOX_TEST_VM_ID"); idStr != "" {
		parsed, err := strconv.ParseInt(idStr, 10, 64)
		require.NoError(t, err)
		vmID = parsed
	} else if diskPath := os.Getenv("FREEBOX_TEST_DISK_PATH"); diskPath != "" {
		name := "pulumi-test-power-" + uuid.New().String()[:8]
		urn := virtualMachineURN(name)
		createResp, err := server.Create(p.CreateRequest{
			Urn: urn,
			Properties: property.NewMap(map[string]property.Value{
				"name":              property.New(name),
				"diskPath":          property.New(diskPath),
				"diskType":          property.New("qcow2"),
				"memory":            property.New(float64(256)),
				"vcpus":             property.New(float64(1)),
				"status":            property.New("stopped"),
				"enableCloudinit":   property.New(false),
				"cloudinitUserdata": property.New(""),
				"cloudinitHostname": property.New(""),
			}),
		})
		require.NoError(t, err)
		var err2 error
		vmID, err2 = strconv.ParseInt(createResp.ID, 10, 64)
		require.NoError(t, err2)
		ownsVM = true
		cleanupVM = func() {
			_ = server.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn, Properties: createResp.Properties})
		}
	} else {
		t.Skip("FREEBOX_TEST_VM_ID or FREEBOX_TEST_DISK_PATH required for VirtualMachinePower integration tests")
	}
	if cleanupVM != nil {
		t.Cleanup(cleanupVM)
	}

	t.Run("ensure stopped, read and delete", func(t *testing.T) {
		vm, err := freeboxClient.GetVirtualMachine(ctx, vmID)
		require.NoError(t, err)
		if vm.Status != freeboxTypes.StoppedStatus {
			require.NoError(t, freeboxClient.StopVirtualMachine(ctx, vmID))
		}

		powerName := "power-" + uuid.New().String()[:8]
		urn := virtualMachinePowerURN(powerName)
		createResp, err := server.Create(p.CreateRequest{
			Urn: urn,
			Properties: property.NewMap(map[string]property.Value{
				"vmId":       property.New(float64(vmID)),
				"powerState": property.New("stopped"),
			}),
		})
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(vmID, 10), createResp.ID)

		readResp, err := server.Read(p.ReadRequest{
			ID:  createResp.ID,
			Urn: urn,
			Properties: property.NewMap(map[string]property.Value{
				"vmId":       property.New(float64(vmID)),
				"powerState": property.New("stopped"),
			}),
		})
		require.NoError(t, err)
		assert.Equal(t, property.New("stopped"), readResp.Properties.Get("powerState"))

		err = server.Delete(p.DeleteRequest{
			ID:         createResp.ID,
			Urn:        urn,
			Properties: createResp.Properties,
		})
		require.NoError(t, err)

		vm, err = freeboxClient.GetVirtualMachine(ctx, vmID)
		require.NoError(t, err)
		if ownsVM {
			assert.Equal(t, freeboxTypes.StoppedStatus, vm.Status)
		}
	})
}
