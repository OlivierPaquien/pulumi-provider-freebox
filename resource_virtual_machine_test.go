// Integration tests for freebox:virtual:Machine.
// Run only when FREEBOX_TEST_DISK_PATH is set (path to an existing qcow2 on the Freebox, e.g. Freebox/VMs/alpine.qcow2).
package main

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/nikolalohinski/free-go/client"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const virtualMachineType = "freebox:virtual:Machine"

func virtualMachineURN(name string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type(virtualMachineType), name)
}

func testVirtualMachine(t *testing.T) {
	ctx := context.Background()
	cfg := freeboxTestConfig()
	server := newTestServer(t, cfg)
	freeboxClient := newFreeboxClient(t, ctx)

	diskPath := os.Getenv("FREEBOX_TEST_DISK_PATH")
	require.NotEmpty(t, diskPath, "FREEBOX_TEST_DISK_PATH must be set for VM tests")

	resourceName := "pulumi-test-vm-" + uuid.New().String()[:8]

	t.Run("create and delete", func(t *testing.T) {
		urn := virtualMachineURN(resourceName)
		// Create stopped to avoid long boot; delete will work in any case.
		props := property.NewMap(map[string]property.Value{
			"name":              property.New(resourceName),
			"diskPath":          property.New(diskPath),
			"diskType":          property.New("qcow2"),
			"memory":            property.New(float64(256)),
			"vcpus":             property.New(float64(1)),
			"status":            property.New("stopped"),
			"enableCloudinit":   property.New(false),
			"cloudinitUserdata": property.New(""),
			"cloudinitHostname": property.New(""),
		})

		createResp, err := server.Create(p.CreateRequest{Urn: urn, Properties: props})
		require.NoError(t, err)
		require.NotEmpty(t, createResp.ID)

		vmID, err := strconv.ParseInt(createResp.ID, 10, 64)
		require.NoError(t, err)

		vm, err := freeboxClient.GetVirtualMachine(ctx, vmID)
		require.NoError(t, err)
		require.NotNil(t, vm)
		assert.Equal(t, resourceName, vm.Name)
		assert.Equal(t, int64(256), vm.Memory)
		assert.Equal(t, int64(1), vm.VCPUs)

		err = server.Delete(p.DeleteRequest{ID: createResp.ID, Urn: urn, Properties: createResp.Properties})
		require.NoError(t, err)

		_, err = freeboxClient.GetVirtualMachine(ctx, vmID)
		assert.ErrorIs(t, err, client.ErrVirtualMachineNotFound)
	})
}
